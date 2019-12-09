package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	cid "gx/ipfs/QmTbxNB1NwDesLmKTscr4udL2tVP7MaxvXnD1D9yX7g3PN/go-cid"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"

	"github.com/OpenBazaar/jsonpb"
	"github.com/OpenBazaar/openbazaar-go/ipfs"
	"github.com/OpenBazaar/openbazaar-go/pb"
	"github.com/OpenBazaar/openbazaar-go/util"
	"github.com/golang/protobuf/proto"
	"github.com/gosimple/slug"
	"github.com/microcosm-cc/bluemonday"
)

const (
	// ListingVersion - current listing version
	ListingVersion = 5
	// TitleMaxCharacters - max size for title
	TitleMaxCharacters = 140
	// ShortDescriptionLength - min length for description
	ShortDescriptionLength = 160
	// DescriptionMaxCharacters - max length for description
	DescriptionMaxCharacters = 50000
	// MaxTags - max permitted tags
	MaxTags = 10
	// MaxCategories - max permitted categories
	MaxCategories = 10
	// MaxListItems - max items in a listing
	MaxListItems = 30
	// FilenameMaxCharacters - max filename size
	FilenameMaxCharacters = 255
	// CodeMaxCharacters - max chars for a code
	CodeMaxCharacters = 20
	// WordMaxCharacters - max chars for word
	WordMaxCharacters = 40
	// SentenceMaxCharacters - max chars for sentence
	SentenceMaxCharacters = 70
	// CouponTitleMaxCharacters - max length of a coupon title
	CouponTitleMaxCharacters = 70
	// PolicyMaxCharacters - max length for policy
	PolicyMaxCharacters = 10000
	// AboutMaxCharacters - max length for about
	AboutMaxCharacters = 10000
	// URLMaxCharacters - max length for URL
	URLMaxCharacters = 2000
	// MaxCountryCodes - max country codes
	MaxCountryCodes = 255
	// DefaultEscrowTimeout - escrow timeout in hours
	DefaultEscrowTimeout = 1080
	// SlugBuffer - buffer size for slug
	SlugBuffer = 5
	// PriceModifierMin - min price modifier
	PriceModifierMin = -99.99
	// PriceModifierMax = max price modifier
	PriceModifierMax = 1000.00
)

type option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type options []option

func (os options) ToOrderOptionSetProtobuf() []*pb.Order_Item_Option {
	var optionProtos = make([]*pb.Order_Item_Option, len(os), len(os))
	for i, o := range os {
		optionProtos[i] = &pb.Order_Item_Option{
			Name:  o.Name,
			Value: o.Value,
		}
	}
	return optionProtos
}

type shippingOption struct {
	Name    string `json:"name"`
	Service string `json:"service"`
}

type Item struct {
	ListingHash    string         `json:"listingHash"`
	Quantity       string         `json:"bigQuantity"`
	Options        options        `json:"options"`
	Shipping       shippingOption `json:"shipping"`
	Memo           string         `json:"memo"`
	Coupons        []string       `json:"coupons"`
	PaymentAddress string         `json:"paymentAddress"`
}

// PurchaseData - record purchase data
type PurchaseData struct {
	ShipTo               string  `json:"shipTo"`
	Address              string  `json:"address"`
	City                 string  `json:"city"`
	State                string  `json:"state"`
	PostalCode           string  `json:"postalCode"`
	CountryCode          string  `json:"countryCode"`
	AddressNotes         string  `json:"addressNotes"`
	Moderator            string  `json:"moderator"`
	Items                []Item  `json:"items"`
	AlternateContactInfo string  `json:"alternateContactInfo"`
	RefundAddress        *string `json:"refundAddress"` //optional, can be left out of json
	PaymentCoin          string  `json:"paymentCoin"`
}

// IndividualListingContainer -
type IndividualListingContainer struct {
	Listing `json:"listing"`
}

// Listing represents a trade offer which can be accepted by another
// party on the OpenBazaar network
type Listing struct {
	listingProto *pb.Listing

	proto.Message
}

func (l *Listing) Reset()         { *l = Listing{} }
func (l *Listing) String() string { return proto.CompactTextString(l) }
func (*Listing) ProtoMessage()    {}

// NewListingFromProtobuf - return Listing from pb.Listing
func NewListingFromProtobuf(l *pb.Listing) (*Listing, error) {
	if l.Metadata.Version == 0 {
		l.Metadata.Version = ListingVersion
	}
	if l.Metadata.EscrowTimeoutHours == 0 {
		l.Metadata.EscrowTimeoutHours = DefaultEscrowTimeout
	}
	return &Listing{
		listingProto: l,
	}, nil
}

// CreateListing will create a pb Listing
func CreateListing(r []byte, isTestnet bool, dstore *Datastore, repoPath string) (Listing, error) {
	ld := new(pb.Listing)
	err := jsonpb.UnmarshalString(string(r), ld)
	if err != nil {
		return Listing{}, err
	}
	slug := ld.Slug
	exists, err := listingExists(slug, repoPath, isTestnet)
	if err != nil {
		return Listing{}, err
	}
	if exists {
		return Listing{}, ErrListingAlreadyExists
	}
	if slug == "" {
		slug, err = GenerateSlug(ld.Item.Title, repoPath, isTestnet, dstore)
		if err != nil {
			return Listing{}, err
		}
		ld.Slug = slug
	}
	retListing, err := NewListingFromProtobuf(ld)

	return *retListing, err
}

// UpdateListing will update a pb Listing
func UpdateListing(r []byte, isTestnet bool, dstore *Datastore, repoPath string) (Listing, error) {
	ld := new(pb.Listing)
	err := jsonpb.UnmarshalString(string(r), ld)
	if err != nil {
		return Listing{}, err
	}
	slug := ld.Slug
	exists, err := listingExists(slug, repoPath, isTestnet)
	if err != nil {
		return Listing{}, err
	}
	if !exists {
		return Listing{}, ErrListingDoesNotExist
	}
	retListing, err := NewListingFromProtobuf(ld)
	return *retListing, err
}

func listingExists(slug, repoPath string, isTestnet bool) (bool, error) {
	if slug == "" {
		return false, nil
	}
	fPath, err := GetPathForListingSlug(slug, repoPath, isTestnet)
	if err != nil {
		return false, err
	}
	_, ferr := os.Stat(fPath)
	if slug == "" {
		return false, nil
	}
	if os.IsNotExist(ferr) {
		return false, nil
	}
	if ferr != nil {
		return false, ferr
	}
	return true, nil
}

// GenerateSlug - slugify the title of the listing
func GenerateSlug(title, repoPath string, isTestnet bool, dStore *Datastore) (string, error) {
	title = strings.Replace(title, "/", "", -1)
	counter := 1
	slugBase := CreateSlugFor(title)
	slugToTry := slugBase
	for {
		_, err := GetListingFromSlug(slugToTry, repoPath, isTestnet, dStore)
		if os.IsNotExist(err) {
			return slugToTry, nil
		} else if err != nil {
			return "", err
		}
		slugToTry = slugBase + strconv.Itoa(counter)
		counter++
	}
}

// GetListingFromSlug - fetch listing for the specified slug
func GetListingFromSlug(slug, repoPath string, isTestnet bool, dStore *Datastore) (*pb.SignedListing, error) {
	repoPath, err := GetRepoPath(isTestnet, repoPath)
	if err != nil {
		return nil, err
	}
	// Read listing file
	listingPath := path.Join(repoPath, "root", "listings", slug+".json")
	file, err := ioutil.ReadFile(listingPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal listing
	sl := new(pb.SignedListing)
	err = jsonpb.UnmarshalString(string(file), sl)
	if err != nil {
		return nil, err
	}

	// Get the listing inventory
	inventory, err := (*dStore).Inventory().Get(slug)
	if err != nil {
		return nil, err
	}

	// Build the inventory list
	for variant, count := range inventory {
		for i, s := range sl.Listing.Item.Skus {
			if variant == i {
				s.BigQuantity = count.String()
				break
			}
		}
	}
	return sl, nil
}

func GetPathForListingSlug(slug, repoPath string, isTestnet bool) (string, error) {
	repoPath, err := GetRepoPath(isTestnet, repoPath)
	if err != nil {
		return "", err
	}
	return path.Join(repoPath, "root", "listings", slug+".json"), nil
}

func ToHtmlEntities(str string) string {
	var rx = regexp.MustCompile(util.EmojiPattern)
	return rx.ReplaceAllStringFunc(str, func(s string) string {
		r, _ := utf8.DecodeRuneInString(s)
		html := fmt.Sprintf(`&#x%X;`, r)
		return html
	})
}

// CreateSlugFor Create a slug from a multi-lang string
func CreateSlugFor(slugName string) string {
	l := SentenceMaxCharacters - SlugBuffer

	slugName = ToHtmlEntities(slugName)

	slug := slug.Make(slugName)
	if len(slug) < SentenceMaxCharacters-SlugBuffer {
		l = len(slug)
	}
	return slug[:l]
}

//func (r Listing) eof() bool {
//return len(r.ListingBytes) == 0
//}

//func (r *Listing) readByte(n int) byte {
//// this function assumes that eof() check was done before
//return r.ListingBytes[n]
//}

//func (r *Listing) Read(p []byte) (n int, err error) {
//if n == len(r.ListingBytes)-1 {
//return
//}

//if c := len(r.ListingBytes); c > 0 {
//for n < c {
//p[n] = r.readByte(n)
//n++
//if r.eof() {
//break
//}
//}
//}
//return n, nil
//}

// ListingMetadata -
type ListingMetadata struct {
	Version uint `json:"version"`
}

// UnmarshalJSONListing - unmarshal listing
func UnmarshalJSONListing(data []byte) (*Listing, error) {
	l, err := UnmarshalJSONSignedListing(data)
	if err != nil {
		return nil, err
	}
	return l.GetListing(), nil
}

// ExtractVersion returns the version of the listing
func ExtractVersionFromSignedListing(data []byte) (uint, error) {
	type sl struct {
		Listing interface{} `json:"listing"`
	}
	var s sl
	err := json.Unmarshal(data, &s)

	if err != nil {
		return 0, err
	}

	lmap, ok := s.Listing.(map[string]interface{})
	if !ok {
		return 0, errors.New("malformed listing")
	}

	lampMeta0, ok := lmap["metadata"]
	if !ok {
		return 0, errors.New("malformed listing")
	}

	lampMeta, ok := lampMeta0.(map[string]interface{})
	if !ok {
		return 0, errors.New("malformed listing")
	}

	ver0, ok := lampMeta["version"]
	if !ok {
		return 0, errors.New("malformed listing")
	}

	ver, ok := ver0.(float64)
	if !ok {
		return 0, errors.New("malformed listing")
	}

	return uint(ver), nil
}

// ExtractIDFromListing returns pb.ID of the listing
func ExtractIDFromListing(data []byte) (*pb.ID, error) {
	var lmap map[string]*json.RawMessage
	vendorPlay := new(pb.ID)

	err := json.Unmarshal(data, &lmap)
	if err != nil {
		log.Error(err)
		return vendorPlay, err
	}

	err = json.Unmarshal(*lmap["vendorID"], &vendorPlay)
	if err != nil {
		log.Error(err)
		return vendorPlay, err
	}

	return vendorPlay, nil
}

// GetProtobuf returns the current state of pb.Listing managed by Listing
func (l *Listing) GetProtobuf() *pb.Listing {
	return l.listingProto
}

// GetVersion returns the schema version of the Listing
func (l *Listing) GetVersion() uint32 {
	return l.listingProto.Metadata.Version
}

// GetCryptoDivisibility returns the listing crypto divisibility
func (l *Listing) GetCryptoDivisibility() uint32 {
	if l.GetContractType() != pb.Listing_Metadata_CRYPTOCURRENCY.String() {
		return 0
	}

	switch l.GetVersion() {
	case 5:
		return l.listingProto.Metadata.CryptoDivisibility
	default: // version < 4
		div := l.listingProto.Metadata.CryptoDivisibility
		if div != 0 {
			div = uint32(math.Log10(float64(div)))
		}
		return div
	}
	return 0
}

// GetCryptoCurrencyCode returns the listing crypto currency code
func (l *Listing) GetCryptoCurrencyCode() string {
	if l.GetContractType() != pb.Listing_Metadata_CRYPTOCURRENCY.String() {
		return ""
	}

	return l.listingProto.Metadata.CryptoCurrencyCode
}

// GetTitle returns the listing item title
func (l *Listing) GetTitle() string {
	return l.listingProto.Item.Title
}

// GetSlug returns the listing slug
func (l *Listing) GetSlug() string {
	return l.listingProto.Slug
}

// GetAcceptedCurrencies returns the listing's list of accepted currency codes
func (l *Listing) GetAcceptedCurrencies() []string {
	return l.listingProto.Metadata.AcceptedCurrencies
}

// SetAcceptedCurrencies the listing's accepted currency codes. Assumes the node
// serving the listing has already validated the wallet supports the currencies.
func (l *Listing) SetAcceptedCurrencies(codes ...string) error {
	if len(codes) < 1 {
		return errors.New("no accepted currencies provided")
	}
	var accCurrencies = make([]string, 0)
	for _, c := range codes {
		def, err := AllCurrencies().Lookup(c)
		if err != nil {
			return fmt.Errorf("unknown accepted currency (%s)", c)
		}
		accCurrencies = append(accCurrencies, def.CurrencyCode().String())
	}
	l.listingProto.Metadata.AcceptedCurrencies = accCurrencies
	return nil
}

// GetContractType returns listing's contract type
func (l *Listing) GetContractType() string {
	return l.listingProto.Metadata.ContractType.String()
}

// GetFormat returns the listing's pricing format
func (l *Listing) GetFormat() string {
	return l.listingProto.Metadata.Format.String()
}

// GetPrice returns the listing price. For CRYPTOCURRENCY listings, this
// value would be zero in the denomination of the cryptocurrency being
// traded and the item value in all other cases. In the event that the
// shema version or contract type are unrecognizable, an error is returned.
func (l *Listing) GetPrice() (*CurrencyValue, error) {
	switch l.GetContractType() {
	case pb.Listing_Metadata_CRYPTOCURRENCY.String():
		return &CurrencyValue{
			Amount:   big.NewInt(0),
			Currency: NewUnknownCryptoDefinition(l.GetCryptoCurrencyCode(), 0),
		}, nil
	case pb.Listing_Metadata_DIGITAL_GOOD.String(), pb.Listing_Metadata_PHYSICAL_GOOD.String(), pb.Listing_Metadata_SERVICE.String():
		switch l.GetVersion() {
		case 5:
			return NewCurrencyValueFromProtobuf(l.listingProto.Item.BigPrice, l.listingProto.Item.PriceCurrency)
		case 4, 3, 2:
			priceCurrency, err := AllCurrencies().Lookup(l.listingProto.Metadata.PricingCurrency)
			if err != nil {
				return nil, fmt.Errorf("lookup metadata pricing currency: %s", err)
			}
			return NewCurrencyValueFromUint(l.listingProto.Item.Price, priceCurrency)
		default:
			return nil, fmt.Errorf("unknown schema version")
		}
	}
	return nil, fmt.Errorf("unknown contract type")
}

// GetModerators returns accepted moderators for the listing
func (l *Listing) GetModerators() []string {
	return l.listingProto.Moderators
}

// SetModerators updates the listing's accepted moderators
func (l *Listing) SetModerators(mods []string) error {
	l.listingProto.Moderators = mods
	// mutations should return an error, even if no error is possible today
	return nil
}

// GetTermsAndConditions return the terms for the listings purchase contract
func (l *Listing) GetTermsAndConditions() string {
	return l.listingProto.TermsAndConditions
}

// GetRefundPolicy return the refund policy for the listing
func (l *Listing) GetRefundPolicy() string {
	return l.listingProto.RefundPolicy
}

// GetVendorID returns the vendor peer ID
func (l *Listing) GetVendorID() *PeerInfo {
	return NewPeerInfoFromProtobuf(l.listingProto.VendorID)
}

// GetShortDescription returns the item description truncated down to the
// ShortDescriptionLength maximum
func (l *Listing) GetShortDescription() string {
	dl := len(l.GetDescription())
	if dl > ShortDescriptionLength {
		return l.GetDescription()[:ShortDescriptionLength]
	}
	return l.GetDescription()
}

// GetDescription returns item description
func (l *Listing) GetDescription() string {
	return l.listingProto.Item.Description
}

// GetProcessingTime returns the expected time for vendor to process listing fulfillment
func (l *Listing) GetProcessingTime() string {
	return l.listingProto.Item.ProcessingTime
}

// GetNSFW returns whether the listing is marked as inappropriate for general viewing
// or otherwise "Not Safe For Work"
func (l *Listing) GetNsfw() bool {
	return l.listingProto.Item.Nsfw
}

// GetTags returns a list of tags for the listing
func (l *Listing) GetTags() []string {
	return l.listingProto.Item.Tags
}

// GetCategories returns a list of categories for the listing
func (l *Listing) GetCategories() []string {
	return l.listingProto.Item.Categories
}

// GetGrams returns listing item weight in grams
func (l *Listing) GetWeightGrams() float32 {
	return l.listingProto.Item.Grams
}

// GetCondition returns listing item condition
func (l *Listing) GetCondition() string {
	return l.listingProto.Item.Condition
}

// ListingImage represents the underlying protobuf image
type ListingImage struct {
	listing    *Listing
	protoIndex int

	filename string
	original string
	large    string
	medium   string
	small    string
	tiny     string
}

func (i *ListingImage) Filename() string { return i.filename }
func (i *ListingImage) Original() string { return i.original }
func (i *ListingImage) Large() string    { return i.large }
func (i *ListingImage) Medium() string   { return i.medium }
func (i *ListingImage) Small() string    { return i.small }
func (i *ListingImage) Tiny() string     { return i.tiny }
func (i *ListingImage) String() string   { return i.Filename() }

func (i *ListingImage) imageProtobuf() (*pb.Listing_Item_Image, error) {
	if i == nil ||
		i.listing == nil ||
		i.listing.listingProto == nil ||
		i.listing.listingProto.Item == nil ||
		i.listing.listingProto.Item.Images == nil {
		return nil, fmt.Errorf("listing item image incomplete")
	}
	pbImg := i.listing.listingProto.Item.Images[i.protoIndex]
	if i.filename != pbImg.GetFilename() ||
		i.original != pbImg.GetOriginal() ||
		i.large != pbImg.GetLarge() ||
		i.medium != pbImg.GetMedium() ||
		i.small != pbImg.GetSmall() ||
		i.tiny != pbImg.GetTiny() {
		return nil, fmt.Errorf("underlying protobuf has changed from expected state")
	}
	return pbImg, nil
}

func (i *ListingImage) SetOriginal(cid string) error {
	var pbi, err = i.imageProtobuf()
	if err != nil {
		return fmt.Errorf("set original image hash: %s", err.Error())
	}
	pbi.Original = cid
	i.original = cid
	return nil
}

func (i *ListingImage) SetLarge(cid string) error {
	var pbi, err = i.imageProtobuf()
	if err != nil {
		return fmt.Errorf("set large image hash: %s", err.Error())
	}
	pbi.Large = cid
	i.large = cid
	return nil
}

func (i *ListingImage) SetMedium(cid string) error {
	var pbi, err = i.imageProtobuf()
	if err != nil {
		return fmt.Errorf("set medium image hash: %s", err.Error())
	}
	pbi.Medium = cid
	i.medium = cid
	return nil
}

func (i *ListingImage) SetSmall(cid string) error {
	var pbi, err = i.imageProtobuf()
	if err != nil {
		return fmt.Errorf("set small image hash: %s", err.Error())
	}
	pbi.Small = cid
	i.small = cid
	return nil
}

func (i *ListingImage) SetTiny(cid string) error {
	var pbi, err = i.imageProtobuf()
	if err != nil {
		return fmt.Errorf("set tiny image hash: %s", err.Error())
	}
	pbi.Tiny = cid
	i.tiny = cid
	return nil
}

// GetImages return set of listing item images
func (l *Listing) GetImages() []*ListingImage {
	if l == nil ||
		l.listingProto == nil ||
		l.listingProto.Item == nil ||
		l.listingProto.Item.Images == nil {
		return make([]*ListingImage, 0, 0)
	}
	var (
		protoImgs = l.listingProto.Item.Images
		imgs      = make([]*ListingImage, len(protoImgs), len(protoImgs))
	)
	for i, img := range protoImgs {
		imgs[i] = &ListingImage{
			listing:    l,
			protoIndex: i,
			filename:   img.Filename,
			original:   img.Original,
			large:      img.Large,
			medium:     img.Medium,
			small:      img.Small,
			tiny:       img.Tiny,
		}
	}
	return imgs
}

//type ListingVariants []ListingVariant

//// ListingSKU represents a unique listing variant
//type ListingVariant struct {
//skuIndex  int
//sku       string
//surcharge *CurrencyValue
//quantity  *big.Int
//}

//// GetVariants returns all unique SKUs and their options
//func (l *Listing) GetVariants() (ListingVariants, error) {
//var (
//schemaVersion = l.GetVersion()
//lvs           = make([]ListingVariant, len(l.listingProto.Item.Skus), len(l.listingProto.Item.Skus))
//skuPrice, err = l.GetPrice()
//)
//if err != nil {
//return nil, fmt.Errorf("get listing price for sku: %s", err.Error())
//}
//for i, s := range l.listingProto.Item.Skus {
//switch schemaVersion {
//case 5:
//var (
//skuQ, ok   = new(big.Int).SetString(s.BigQuantity, 10)
//skuCV, err = NewCurrencyValue(s.BigSurcharge, skuPrice.Currency)
//)
//if !ok {
//return nil, fmt.Errorf("parsing sku (%s) quantity: %s", s.ProductID, err.Error())
//}
//if err != nil {
//return nil, fmt.Errorf("parsing sku (%s) surcharge: %s", s.ProductID, err.Error())
//}

//lvs[i] = ListingVariant{
//skuIndex:  i,
//sku:       s.ProductID,
//surcharge: skuCV,
//quantity:  skuQ,
//}
//case 4, 3, 2:
//var skuCV, err = NewCurrencyValueFromInt(s.Surcharge, skuPrice.Currency)
//if err != nil {
//return nil, fmt.Errorf("parsing sku (%s) surcharge: %s", s.ProductID, err.Error())
//}

//lvs[i] = ListingVariant{
//skuIndex:  i,
//sku:       s.ProductID,
//surcharge: skuCV,
//quantity:  new(big.Int).SetInt64(s.Quantity),
//}
//}
//}
//return lvs, nil
//}

// GetSkus returns the listing SKUs
func (l *Listing) GetSkus() ([]*pb.Listing_Item_Sku, error) {
	switch l.GetVersion() {
	case 3, 4:
		for _, sku := range l.listingProto.Item.Skus {
			surcharge := new(big.Int).SetInt64(sku.Surcharge)
			quantity := new(big.Int).SetInt64(sku.Quantity)
			sku.BigSurcharge = surcharge.String()
			sku.BigQuantity = quantity.String()
			sku.Quantity = 0
			sku.Surcharge = 0
		}
	}
	return l.listingProto.Item.Skus, nil
}

// GetItem - return item
//func (l *Listing) GetItem() (*pb.Listing_Item, error) {
//title, err := l.GetTitle()
//if err != nil {
//return nil, err
//}
//description, err := l.GetDescription()
//if err != nil {
//return nil, err
//}
//processingtime, err := l.GetProcessingTime()
//if err != nil {
//return nil, err
//}
//nsfw, err := l.GetNsfw()
//if err != nil {
//return nil, err
//}
//tags, err := l.GetTags()
//if err != nil {
//return nil, err
//}
//images, err := l.GetImages()
//if err != nil {
//return nil, err
//}
//categories, err := l.GetCategories()
//if err != nil {
//return nil, err
//}
//grams, err := l.GetGrams()
//if err != nil {
//return nil, err
//}
//condition, err := l.GetCondition()
//if err != nil {
//return nil, err
//}
//options, err := l.GetOptions()
//if err != nil {
//return nil, err
//}
//skus, err := l.GetSkus()
//if err != nil {
//return nil, err
//}
//price, err := l.GetPrice()
//if err != nil {
//return nil, err
//}
//i := pb.Listing_Item{
//Title:          title,
//Description:    description,
//ProcessingTime: processingtime,
//Nsfw:           nsfw,
//Tags:           tags,
//Images:         images,
//Categories:     categories,
//Grams:          grams,
//Condition:      condition,
//Options:        options,
//Skus:           skus,
//BigPrice:       price.Amount.String(),
//PriceCurrency: &pb.CurrencyDefinition{
//Code:         price.Currency.Code.String(),
//Divisibility: uint32(price.Currency.Divisibility),
//},
//}
//return &i, nil
//}

// GetExpiry return listing expiry
//func (l *Listing) GetExpiry() (*timestamp.Timestamp, error) {
//type expiry struct {
//Metadata struct {
//Expiry string `json:"expiry"`
//} `json:"metadata"`
//}
//var exp expiry
//err := json.Unmarshal(l.ListingBytes, &exp)
//if err != nil {
//return nil, err
//}
//t := new(timestamp.Timestamp)

//t0, err := time.Parse(time.RFC3339Nano, exp.Metadata.Expiry)
//if err != nil {
//return nil, err
//}

//t.Seconds = t0.Unix()
//t.Nanos = int32(t0.Nanosecond())

//return t, nil
//}

//GetLanguage return listing's language
func (l *Listing) GetLanguage() string {
	return l.listingProto.Metadata.Language
}

// GetEscrowTimeout return listing's escrow timeout in hours
func (l *Listing) GetEscrowTimeoutHours() uint32 {
	return l.listingProto.Metadata.EscrowTimeoutHours
}

// GetPriceModifier return listing's price modifier
func (l *Listing) GetPriceModifier() float32 {
	switch l.GetVersion() {
	case 5:
		return l.listingProto.Item.PriceModifier
	case 4, 3, 2:
		return l.listingProto.Metadata.PriceModifier
	}
	log.Errorf("missing price modifier for listing (%s)", l.GetSlug())
	return 0
}

type ListingTaxes []ListingTax

type ListingTax struct {
	taxType         string
	regions         []string
	rate            float32
	taxableShipping bool
}

func (t ListingTax) Type() string                { return t.taxType }
func (t ListingTax) ApplicableRegions() []string { return t.regions }
func (t ListingTax) Rate() float32               { return t.rate }
func (t ListingTax) TaxableShipping() bool       { return t.taxableShipping }

// GetTaxes returns listing tax information
func (l *Listing) GetTaxes() ListingTaxes {
	var ts = make([]ListingTax, len(l.listingProto.Taxes), len(l.listingProto.Taxes))
	for ti, tax := range l.listingProto.Taxes {
		var rs = make([]string, len(tax.TaxRegions), len(tax.TaxRegions))
		for ri, region := range tax.TaxRegions {
			rs[ri] = region.String()
		}

		ts[ti] = ListingTax{
			taxType:         tax.TaxType,
			rate:            tax.Percentage,
			taxableShipping: tax.TaxShipping,
			regions:         rs,
		}
	}
	return ts
}

type couponGetter interface {
	Get(string) ([]Coupon, error)
}

// UpdateCouponsFromDatastore will get all coupons from the datastore and update
// the internal protobuf with the codes that match the coupon's hash, if any.
func (l *Listing) UpdateCouponsFromDatastore(cdb couponGetter) error {
	var coupons, err = l.GetCoupons()
	if err != nil {
		return fmt.Errorf("getting coupons: %s", err.Error())
	}
	dbCoupons, err := cdb.Get(l.GetSlug())
	if err != nil {
		return fmt.Errorf("loading datastore coupon: %s", err.Error())
	}
	for _, c := range coupons {
		for _, dbc := range dbCoupons {
			if c.redemptionHash == dbc.Hash {
				// make sure applying code does not shift already-matched hash
				expectedHash, err := ipfs.EncodeMultihash([]byte(dbc.Code))
				if err != nil {
					return fmt.Errorf("hashing persisted redemption code (%s): %s", dbc.Code, err.Error())
				}
				if c.redemptionHash != expectedHash.B58String() {
					return fmt.Errorf("update coupon code (%s) results in mismatched published hash", dbc.Code)
				}
				if err := c.SetRedemptionCode(dbc.Code); err != nil {
					return fmt.Errorf("setting redemption code: %s", err.Error())
				}
			}
		}
	}
	return nil
}

// GetCoupons returns listing coupons with discount amount normalized as a
// CurrencyValue
func (l *Listing) GetCoupons() (ListingCoupons, error) {
	var (
		protoCoupons   = l.listingProto.GetCoupons()
		cs             = make([]*ListingCoupon, len(protoCoupons), len(protoCoupons))
		listingVersion = l.GetVersion()
		discPrice, err = l.GetPrice()
	)
	if err != nil {
		return nil, fmt.Errorf("get listing price for coupon: %s", err.Error())
	}

	for i, c := range protoCoupons {
		var discAmt string
		switch listingVersion {
		case 5:
			discAmt = c.GetBigPriceDiscount()
		default:
			discAmt = strconv.FormatUint(c.GetPriceDiscount(), 10)
		}
		discValue, err := NewCurrencyValue(discAmt, discPrice.Currency)
		if err != nil {
			return nil, fmt.Errorf("unable to create coupon discount value for amount (%s %s): %s", discAmt, discPrice.Currency, err.Error())
		}

		cs[i] = &ListingCoupon{
			listing:         l,
			title:           c.GetTitle(),
			redemptionCode:  c.GetDiscountCode(),
			redemptionHash:  c.GetHash(),
			discountPercent: c.GetPercentDiscount(),
			discountAmount:  discValue,
		}
	}
	return cs, nil
}

type ListingCoupons []*ListingCoupon
type ListingCoupon struct {
	listing *Listing

	title          string
	redemptionCode string
	redemptionHash string

	discountPercent float32
	discountAmount  *CurrencyValue
}

func (c *ListingCoupon) ListingSlug() string { return c.listing.GetSlug() }
func (c *ListingCoupon) Title() string       { return c.title }
func (c *ListingCoupon) RedemptionCode() (string, error) {
	if c.redemptionCode != "" {
		return c.redemptionCode, nil
	}
	return "", errors.New("redemption code not set")
}
func (c *ListingCoupon) RedemptionHash() (string, error) {
	_, err := mh.FromB58String(c.redemptionHash)
	if err != nil {
		// if hash is invalid, let's try to produce a new one
		if c.redemptionCode == "" {
			return "", errors.New("hash invalid and code missing")
		}
		if err := c.SetRedemptionCode(c.redemptionCode); err != nil {
			return "", err
		}
	}
	return c.redemptionHash, nil
}
func (c *ListingCoupon) PercentOff() float32       { return c.discountPercent }
func (c *ListingCoupon) AmountOff() *CurrencyValue { return c.discountAmount }

func (c *ListingCoupon) SetRedemptionCode(code string) error {
	newHash, err := ipfs.EncodeMultihash([]byte(code))
	if err != nil {
		return fmt.Errorf("hashing redemption code: %s", err.Error())
	}
	// update proto first, otherwise ListingCoupon hash/code can't
	// be used to match against the correct proto to update
	if err := c.updateProtoHash(newHash.B58String()); err != nil {
		return err
	}
	c.redemptionCode = code
	c.redemptionHash = newHash.B58String()
	return nil
}

func (c *ListingCoupon) updateProtoHash(hash string) error {
	for _, pc := range c.listing.listingProto.Coupons {
		if pc.GetDiscountCode() == c.redemptionCode ||
			pc.GetHash() == c.redemptionHash {
			pc.Code = &pb.Listing_Coupon_Hash{Hash: hash}
			return nil
		}
	}
	return errors.New("unable to update missing coupon proto")
}

// GetShippingRegions returns all region strings for the defined shipping
// services
func (l *Listing) GetShippingRegions() ([]string, []string) {
	var (
		shipsTo        = make(map[string]struct{})
		freeShippingTo = make(map[string]struct{})
	)
	for _, shipOption := range l.listingProto.ShippingOptions {
		for _, shipRegion := range shipOption.Regions {
			shipsTo[shipRegion.String()] = struct{}{}
			for _, shipService := range shipOption.Services {
				servicePrice, ok := new(big.Int).SetString(shipService.BigPrice, 10)
				if ok && servicePrice.Cmp(big.NewInt(0)) == 0 {
					freeShippingTo[shipRegion.String()] = struct{}{}
				}
			}
		}
	}

	var returnShipTo = make([]string, 0)
	for s, _ := range shipsTo {
		returnShipTo = append(returnShipTo, s)
	}
	var returnFreeShipTo = make([]string, 0)
	for s, _ := range freeShippingTo {
		returnFreeShipTo = append(returnFreeShipTo, s)
	}
	return returnShipTo, returnFreeShipTo
}

type listingSigner interface {
	TestNetworkEnabled() bool
	RegressionNetworkEnabled() bool
	GetNodeID() (*pb.ID, error)
	Sign([]byte) ([]byte, error)
}

func (l *Listing) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: false,
		Indent:       "    ",
		OrigName:     false,
	}
	lb, err := m.MarshalToString(l.listingProto)
	if err != nil {
		return nil, err
	}
	return []byte(lb), nil
}

// Sign verifies the Listing and returns a SignedListing
func (l *Listing) Sign(n listingSigner) (*SignedListing, error) {
	var (
		timeout   = l.GetEscrowTimeoutHours()
		isTestnet = n.TestNetworkEnabled() || n.RegressionNetworkEnabled()
	)

	// Temporary hack to work around test env shortcomings
	if isTestnet {
		timeout = 1
	}
	l.listingProto.Metadata.EscrowTimeoutHours = timeout

	// Set inventory to the default as it's not part of the contract
	for _, s := range l.listingProto.Item.Skus {
		s.Quantity = 0
		s.BigQuantity = "0"
	}

	// Check the listing data is correct for continuing
	if err := l.ValidateListing(isTestnet); err != nil {
		return nil, err
	}

	// Sanitize a few critical fields
	sanitizer := bluemonday.UGCPolicy()
	for _, opt := range l.listingProto.Item.Options {
		opt.Name = sanitizer.Sanitize(opt.Name)
		for _, v := range opt.Variants {
			v.Name = sanitizer.Sanitize(v.Name)
		}
	}
	for _, so := range l.listingProto.ShippingOptions {
		so.Name = sanitizer.Sanitize(so.Name)
		for _, serv := range so.Services {
			serv.Name = sanitizer.Sanitize(serv.Name)
		}
	}

	// Add the vendor ID to the listing
	id, err := n.GetNodeID()
	if err != nil {
		return nil, fmt.Errorf("vendor id: %s", err.Error())
	}
	l.listingProto.VendorID = id

	// Sign listing
	serializedListing, err := l.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("serializing listing: %s", err.Error())
	}
	listingSig, err := n.Sign(serializedListing)
	if err != nil {
		return nil, fmt.Errorf("signing listing: %s", err.Error())
	}

	sl := new(pb.SignedListing)
	sl.Listing = l.listingProto
	sl.Signature = listingSig

	return &SignedListing{
		signedListingProto: sl,
	}, nil
}

// ValidateSkus ensures valid SKU state
func (l *Listing) ValidateSkus() error {
	if l.listingProto.Metadata.ContractType == pb.Listing_Metadata_CRYPTOCURRENCY {
		return validateCryptocurrencyQuantity(l.listingProto)
	}
	return nil
}

func validateCryptocurrencyQuantity(listing *pb.Listing) error {
	var checkFn func(*pb.Listing_Item_Sku) error
	switch listing.Metadata.Version {
	case 5:
		checkFn = func(s *pb.Listing_Item_Sku) error {
			if s == nil {
				return fmt.Errorf("cannot validate nil sku")
			}
			if s.BigQuantity == "" {
				return fmt.Errorf("sku quantity empty")
			}
			if ba, ok := new(big.Int).SetString(s.BigQuantity, 10); ok && ba.Cmp(big.NewInt(0)) < 0 {
				return fmt.Errorf("sku quantity cannot be negative")
			}
			return nil
		}
	default:
		checkFn = func(s *pb.Listing_Item_Sku) error {
			if s == nil {
				return fmt.Errorf("cannot validate nil sku")
			}
			if s.Quantity <= 0 {
				return fmt.Errorf("sku quantity zero or less")
			}
			return nil
		}
	}
	for _, sku := range listing.Item.Skus {
		if err := checkFn(sku); err != nil {
			return ErrCryptocurrencySkuQuantityInvalid
		}
	}
	return nil
}

// GetInventory - returns a map of skus and quantityies
func (l *Listing) GetInventory() (map[int]*big.Int, error) {
	inventory := make(map[int]*big.Int)
	for i, s := range l.listingProto.Item.Skus {
		var amtStr string
		switch l.GetVersion() {
		case 5:
			amtStr = s.BigQuantity
		default:
			amtStr = strconv.Itoa(int(s.Quantity))
		}
		amt, ok := new(big.Int).SetString(amtStr, 10)
		if !ok {
			return nil, errors.New("error parsing inventory")
		}
		inventory[i] = amt
	}
	return inventory, nil
}

// ValidateListing ensures all listing state is valid
func (l *Listing) ValidateListing(testnet bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()

	// Slug
	if l.listingProto.Slug == "" {
		return errors.New("slug must not be empty")
	}
	if len(l.listingProto.Slug) > SentenceMaxCharacters {
		return fmt.Errorf("slug is longer than the max of %d", SentenceMaxCharacters)
	}
	if strings.Contains(l.listingProto.Slug, " ") {
		return errors.New("slugs cannot contain spaces")
	}
	if strings.Contains(l.listingProto.Slug, "/") {
		return errors.New("slugs cannot contain file separators")
	}

	// Metadata
	if l.listingProto.Metadata == nil {
		return errors.New("missing required field: Metadata")
	}
	if l.listingProto.Metadata.ContractType > pb.Listing_Metadata_CRYPTOCURRENCY {
		return errors.New("invalid contract type")
	}
	if l.listingProto.Metadata.Format > pb.Listing_Metadata_MARKET_PRICE {
		return errors.New("invalid listing format")
	}
	if l.listingProto.Metadata.Expiry == nil {
		return errors.New("missing required field: Expiry")
	}
	if time.Unix(l.listingProto.Metadata.Expiry.Seconds, 0).Before(time.Now()) {
		return errors.New("listing expiration must be in the future")
	}
	if len(l.listingProto.Metadata.Language) > WordMaxCharacters {
		return fmt.Errorf("language is longer than the max of %d characters", WordMaxCharacters)
	}

	if !testnet && l.listingProto.Metadata.EscrowTimeoutHours != DefaultEscrowTimeout {
		return fmt.Errorf("escrow timeout must be %d hours", DefaultEscrowTimeout)
	}
	if len(l.listingProto.Metadata.AcceptedCurrencies) == 0 {
		return errors.New("at least one accepted currency must be provided")
	}
	if len(l.listingProto.Metadata.AcceptedCurrencies) > MaxListItems {
		return fmt.Errorf("acceptedCurrencies is longer than the max of %d currencies", MaxListItems)
	}
	for _, c := range l.listingProto.Metadata.AcceptedCurrencies {
		if len(c) > WordMaxCharacters {
			return fmt.Errorf("accepted currency is longer than the max of %d characters", WordMaxCharacters)
		}
	}

	// Item
	if l.listingProto.Item == nil {
		return errors.New("no item in listing")
	}
	if l.listingProto.Item.Title == "" {
		return errors.New("listing must have a title")
	}
	if l.listingProto.Metadata.ContractType != pb.Listing_Metadata_CRYPTOCURRENCY && l.listingProto.Item.BigPrice == "0" {
		return errors.New("zero price listings are not allowed")
	}
	if len(l.listingProto.Item.Title) > TitleMaxCharacters {
		return fmt.Errorf("title is longer than the max of %d characters", TitleMaxCharacters)
	}
	if len(l.listingProto.Item.Description) > DescriptionMaxCharacters {
		return fmt.Errorf("description is longer than the max of %d characters", DescriptionMaxCharacters)
	}
	if len(l.listingProto.Item.ProcessingTime) > SentenceMaxCharacters {
		return fmt.Errorf("processing time length must be less than the max of %d", SentenceMaxCharacters)
	}
	if len(l.listingProto.Item.Tags) > MaxTags {
		return fmt.Errorf("number of tags exceeds the max of %d", MaxTags)
	}
	for _, tag := range l.listingProto.Item.Tags {
		if tag == "" {
			return errors.New("tags must not be empty")
		}
		if len(tag) > WordMaxCharacters {
			return fmt.Errorf("tags must be less than max of %d", WordMaxCharacters)
		}
	}
	if len(l.listingProto.Item.Images) == 0 {
		return errors.New("listing must contain at least one image")
	}
	if len(l.listingProto.Item.Images) > MaxListItems {
		return fmt.Errorf("number of listing images is greater than the max of %d", MaxListItems)
	}
	for _, img := range l.listingProto.Item.Images {
		_, err := cid.Decode(img.Tiny)
		if err != nil {
			return errors.New("tiny image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Small)
		if err != nil {
			return errors.New("small image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Medium)
		if err != nil {
			return errors.New("medium image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Large)
		if err != nil {
			return errors.New("large image hashes must be properly formatted CID")
		}
		_, err = cid.Decode(img.Original)
		if err != nil {
			return errors.New("original image hashes must be properly formatted CID")
		}
		if img.Filename == "" {
			return errors.New("image file names must not be nil")
		}
		if len(img.Filename) > FilenameMaxCharacters {
			return fmt.Errorf("image filename length must be less than the max of %d", FilenameMaxCharacters)
		}
	}
	if len(l.listingProto.Item.Categories) > MaxCategories {
		return fmt.Errorf("number of categories must be less than max of %d", MaxCategories)
	}
	for _, category := range l.listingProto.Item.Categories {
		if category == "" {
			return errors.New("categories must not be nil")
		}
		if len(category) > WordMaxCharacters {
			return fmt.Errorf("category length must be less than the max of %d", WordMaxCharacters)
		}
	}

	maxCombos := 1
	variantSizeMap := make(map[int]int)
	optionMap := make(map[string]struct{})
	for i, option := range l.listingProto.Item.Options {
		if _, ok := optionMap[option.Name]; ok {
			return errors.New("option names must be unique")
		}
		if option.Name == "" {
			return errors.New("options titles must not be empty")
		}
		if len(option.Variants) < 2 {
			return errors.New("options must have more than one variants")
		}
		if len(option.Name) > WordMaxCharacters {
			return fmt.Errorf("option title length must be less than the max of %d", WordMaxCharacters)
		}
		if len(option.Description) > SentenceMaxCharacters {
			return fmt.Errorf("option description length must be less than the max of %d", SentenceMaxCharacters)
		}
		if len(option.Variants) > MaxListItems {
			return fmt.Errorf("number of variants is greater than the max of %d", MaxListItems)
		}
		varMap := make(map[string]struct{})
		for _, variant := range option.Variants {
			if _, ok := varMap[variant.Name]; ok {
				return errors.New("variant names must be unique")
			}
			if len(variant.Name) > WordMaxCharacters {
				return fmt.Errorf("variant name length must be less than the max of %d", WordMaxCharacters)
			}
			if variant.Image != nil && (variant.Image.Filename != "" ||
				variant.Image.Large != "" || variant.Image.Medium != "" || variant.Image.Small != "" ||
				variant.Image.Tiny != "" || variant.Image.Original != "") {
				_, err := cid.Decode(variant.Image.Tiny)
				if err != nil {
					return errors.New("tiny image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Small)
				if err != nil {
					return errors.New("small image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Medium)
				if err != nil {
					return errors.New("medium image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Large)
				if err != nil {
					return errors.New("large image hashes must be properly formatted CID")
				}
				_, err = cid.Decode(variant.Image.Original)
				if err != nil {
					return errors.New("original image hashes must be properly formatted CID")
				}
				if variant.Image.Filename == "" {
					return errors.New("image file names must not be nil")
				}
				if len(variant.Image.Filename) > FilenameMaxCharacters {
					return fmt.Errorf("image filename length must be less than the max of %d", FilenameMaxCharacters)
				}
			}
			varMap[variant.Name] = struct{}{}
		}
		variantSizeMap[i] = len(option.Variants)
		maxCombos *= len(option.Variants)
		optionMap[option.Name] = struct{}{}
	}

	if len(l.listingProto.Item.Skus) > maxCombos {
		return errors.New("more skus than variant combinations")
	}
	comboMap := make(map[string]bool)
	for _, sku := range l.listingProto.Item.Skus {
		if maxCombos > 1 && len(sku.VariantCombo) == 0 {
			return errors.New("skus must specify a variant combo when options are used")
		}
		if len(sku.ProductID) > WordMaxCharacters {
			return fmt.Errorf("product ID length must be less than the max of %d", WordMaxCharacters)
		}
		formatted, err := json.Marshal(sku.VariantCombo)
		if err != nil {
			return err
		}
		_, ok := comboMap[string(formatted)]
		if !ok {
			comboMap[string(formatted)] = true
		} else {
			return errors.New("duplicate sku")
		}
		if len(sku.VariantCombo) != len(l.listingProto.Item.Options) {
			return errors.New("incorrect number of variants in sku combination")
		}
		for i, combo := range sku.VariantCombo {
			if int(combo) > variantSizeMap[i] {
				return errors.New("invalid sku variant combination")
			}
		}

	}

	// Taxes
	if len(l.listingProto.Taxes) > MaxListItems {
		return fmt.Errorf("number of taxes is greater than the max of %d", MaxListItems)
	}
	for _, tax := range l.listingProto.Taxes {
		if tax.TaxType == "" {
			return errors.New("tax type must be specified")
		}
		if len(tax.TaxType) > WordMaxCharacters {
			return fmt.Errorf("tax type length must be less than the max of %d", WordMaxCharacters)
		}
		if len(tax.TaxRegions) == 0 {
			return errors.New("tax must specify at least one region")
		}
		if len(tax.TaxRegions) > MaxCountryCodes {
			return fmt.Errorf("number of tax regions is greater than the max of %d", MaxCountryCodes)
		}
		if tax.Percentage == 0 || tax.Percentage > 100 {
			return errors.New("tax percentage must be between 0 and 100")
		}
	}

	// Coupons
	if len(l.listingProto.Coupons) > MaxListItems {
		return fmt.Errorf("number of coupons is greater than the max of %d", MaxListItems)
	}
	for _, coupon := range l.listingProto.Coupons {
		if len(coupon.Title) > CouponTitleMaxCharacters {
			return fmt.Errorf("coupon title length must be less than the max of %d", SentenceMaxCharacters)
		}
		if len(coupon.GetDiscountCode()) > CodeMaxCharacters {
			return fmt.Errorf("coupon code length must be less than the max of %d", CodeMaxCharacters)
		}
		if coupon.GetPercentDiscount() > 100 {
			return errors.New("percent discount cannot be over 100 percent")
		}
		n, ok := new(big.Int).SetString(l.listingProto.Item.BigPrice, 10)
		if !ok {
			return errors.New("price was invalid")
		}
		if coupon.GetBigPriceDiscount() != "" {
			discount0, ok := new(big.Int).SetString(coupon.BigPriceDiscount, 10)
			if !ok {
				return errors.New("coupon discount was invalid")
			}
			if n.Cmp(discount0) < 0 {
				return errors.New("price discount cannot be greater than the item price")
			}
		}
		if coupon.GetPercentDiscount() == 0 && coupon.GetBigPriceDiscount() == "" {
			return errors.New("coupons must have at least one positive discount value")
		}
		if coupon.GetPercentDiscount() != 0 && coupon.GetBigPriceDiscount() != "" {
			return errors.New("coupons must have either a percent discount or a fixed amount discount, but not both")
		}
	}

	// Moderators
	if len(l.listingProto.Moderators) > MaxListItems {
		return fmt.Errorf("number of moderators is greater than the max of %d", MaxListItems)
	}
	for _, moderator := range l.listingProto.Moderators {
		_, err := mh.FromB58String(moderator)
		if err != nil {
			return errors.New("moderator IDs must be multihashes")
		}
	}

	// TermsAndConditions
	if len(l.listingProto.TermsAndConditions) > PolicyMaxCharacters {
		return fmt.Errorf("terms and conditions length must be less than the max of %d", PolicyMaxCharacters)
	}

	// RefundPolicy
	if len(l.listingProto.RefundPolicy) > PolicyMaxCharacters {
		return fmt.Errorf("refund policy length must be less than the max of %d", PolicyMaxCharacters)
	}

	// Type-specific validations
	if l.listingProto.Metadata.ContractType == pb.Listing_Metadata_PHYSICAL_GOOD {
		err := l.validatePhysicalListing()
		if err != nil {
			return err
		}
	} else if l.listingProto.Metadata.ContractType == pb.Listing_Metadata_CRYPTOCURRENCY {
		err := l.ValidateCryptoListing()
		if err != nil {
			return err
		}
	}

	// Non-crypto validations
	if l.listingProto.Metadata.ContractType != pb.Listing_Metadata_CRYPTOCURRENCY {
		if l.listingProto.Item.PriceCurrency == nil {
			return errors.New("pricing currency is missing")
		}
		if priceCurrency, err := AllCurrencies().Lookup(l.listingProto.Item.PriceCurrency.Code); err != nil {
			return errors.New("invalid pricing currency")
		} else {
			if uint(l.listingProto.Item.PriceCurrency.Divisibility) > priceCurrency.Divisibility {
				return errors.New("pricing currency divisibility is too large")
			}
		}
		if _, ok := new(big.Int).SetString(l.listingProto.Item.BigPrice, 10); !ok {
			return errors.New("invalid item price amount")
		}
	}

	// Format-specific validations
	if l.listingProto.Metadata.Format == pb.Listing_Metadata_MARKET_PRICE {
		err := validateMarketPriceListing(l.listingProto)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Listing) validatePhysicalListing() error {
	if len(l.listingProto.Item.Condition) > SentenceMaxCharacters {
		return fmt.Errorf("'Condition' length must be less than the max of %d", SentenceMaxCharacters)
	}
	if len(l.listingProto.Item.Options) > MaxListItems {
		return fmt.Errorf("number of options is greater than the max of %d", MaxListItems)
	}

	// ShippingOptions
	if len(l.listingProto.ShippingOptions) == 0 {
		return errors.New("must be at least one shipping option for a physical good")
	}
	if len(l.listingProto.ShippingOptions) > MaxListItems {
		return fmt.Errorf("number of shipping options is greater than the max of %d", MaxListItems)
	}
	var shippingTitles []string
	for _, shippingOption := range l.listingProto.ShippingOptions {
		if shippingOption.Name == "" {
			return errors.New("shipping option title name must not be empty")
		}
		if len(shippingOption.Name) > WordMaxCharacters {
			return fmt.Errorf("shipping option service length must be less than the max of %d", WordMaxCharacters)
		}
		for _, t := range shippingTitles {
			if t == shippingOption.Name {
				return errors.New("shipping option titles must be unique")
			}
		}
		shippingTitles = append(shippingTitles, shippingOption.Name)
		if shippingOption.Type > pb.Listing_ShippingOption_FIXED_PRICE {
			return errors.New("unknown shipping option type")
		}
		if len(shippingOption.Regions) == 0 {
			return errors.New("shipping options must specify at least one region")
		}
		if err := ValidateShippingRegion(shippingOption); err != nil {
			return fmt.Errorf("invalid shipping option (%s): %s", shippingOption.String(), err.Error())
		}
		if len(shippingOption.Regions) > MaxCountryCodes {
			return fmt.Errorf("number of shipping regions is greater than the max of %d", MaxCountryCodes)
		}
		if len(shippingOption.Services) == 0 && shippingOption.Type != pb.Listing_ShippingOption_LOCAL_PICKUP {
			return errors.New("at least one service must be specified for a shipping option when not local pickup")
		}
		if len(shippingOption.Services) > MaxListItems {
			return fmt.Errorf("number of shipping services is greater than the max of %d", MaxListItems)
		}
		var serviceTitles []string
		for _, option := range shippingOption.Services {
			if option.Name == "" {
				return errors.New("shipping option service name must not be empty")
			}
			if len(option.Name) > WordMaxCharacters {
				return fmt.Errorf("shipping option service length must be less than the max of %d", WordMaxCharacters)
			}
			for _, t := range serviceTitles {
				if t == option.Name {
					return errors.New("shipping option services names must be unique")
				}
			}
			serviceTitles = append(serviceTitles, option.Name)

			if option.EstimatedDelivery == "" {
				return errors.New("shipping option estimated delivery must not be empty")
			}
			if len(option.EstimatedDelivery) > SentenceMaxCharacters {
				return fmt.Errorf("shipping option estimated delivery length must be less than the max of %d", SentenceMaxCharacters)
			}
			if _, ok := new(big.Int).SetString(option.BigPrice, 10); !ok {
				return errors.New("invalid shipping service price amount")
			}
		}
	}

	return nil
}

func (l *Listing) ValidateCryptoListing() error {
	if len(l.listingProto.Metadata.AcceptedCurrencies) != 1 {
		return errors.New("cryptocurrency listing must only have one accepted currency")
	}

	if len(l.listingProto.Coupons) > 0 {
		return ErrCryptocurrencyListingIllegalField("coupons")
	}
	if len(l.listingProto.Item.Options) > 0 {
		return ErrCryptocurrencyListingIllegalField("item.options")
	}
	if len(l.listingProto.ShippingOptions) > 0 {
		return ErrCryptocurrencyListingIllegalField("shippingOptions")
	}
	if len(l.listingProto.Item.Condition) > 0 {
		return ErrCryptocurrencyListingIllegalField("item.condition")
	}
	if l.listingProto.Item.PriceCurrency != nil &&
		len(l.listingProto.Item.PriceCurrency.Code) > 0 {
		return ErrCryptocurrencyListingIllegalField("item.pricingCurrency")
	}
	if len(l.listingProto.Metadata.CryptoCurrencyCode) == 0 {
		return ErrListingCryptoCurrencyCodeInvalid
	}

	cryptoDivisibility := l.GetCryptoDivisibility()
	if cryptoDivisibility == 0 {
		return ErrListingCryptoDivisibilityInvalid
	}
	def := NewUnknownCryptoDefinition(l.listingProto.Metadata.CryptoCurrencyCode, uint(cryptoDivisibility))
	if err := def.Valid(); err != nil {
		return fmt.Errorf("cryptocurrency metadata invalid: %s", err)
	}
	return nil
}

func (l *Listing) SetCryptocurrencyListingDefaults() error {
	l.listingProto.Coupons = []*pb.Listing_Coupon{}
	l.listingProto.Item.Options = []*pb.Listing_Item_Option{}
	l.listingProto.ShippingOptions = []*pb.Listing_ShippingOption{}
	l.listingProto.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
	return nil
}

func validateMarketPriceListing(listing *pb.Listing) error {
	var (
		priceModifier   float32
		roundHundredths = func(f float32) float32 { return float32(int(f*100.0)) / 100.0 }
		n, ok           = new(big.Int).SetString(listing.Item.BigPrice, 10)
	)

	if ok && n.Cmp(big.NewInt(0)) != 0 {
		return ErrMarketPriceListingIllegalField("item.bigPrice")
	}

	if listing.Metadata.PriceModifier != 0 {
		priceModifier = roundHundredths(listing.Metadata.PriceModifier)
		listing.Metadata.PriceModifier = priceModifier
	} else if listing.Item.PriceModifier != 0 {
		priceModifier = roundHundredths(listing.Item.PriceModifier)
		listing.Item.PriceModifier = priceModifier
	}

	if priceModifier < PriceModifierMin ||
		priceModifier > PriceModifierMax {
		return ErrPriceModifierOutOfRange{
			Min: PriceModifierMin,
			Max: PriceModifierMax,
		}
	}

	return nil
}

func ValidateShippingRegion(shippingOption *pb.Listing_ShippingOption) error {
	for _, region := range shippingOption.Regions {
		if int32(region) == 0 {
			return ErrShippingRegionMustBeSet
		}
		_, ok := proto.EnumValueMap("CountryCode")[region.String()]
		if !ok {
			return ErrShippingRegionUndefined
		}
		if ok {
			if int32(region) > 500 {
				return ErrShippingRegionMustNotBeContinent
			}
		}
	}
	return nil
}

func (l *Listing) ValidatePurchaseItemOptions(itemOptions []option) error {
	if l.GetContractType() == pb.Listing_Metadata_CRYPTOCURRENCY.String() &&
		len(itemOptions) > 0 {
		return fmt.Errorf("options on a %s listing were provided, but are not supported", pb.Listing_Metadata_CRYPTOCURRENCY.String())
	}

	var (
		optSet     = make(map[string]struct{})
		checkedOpt = make(map[string]struct{})
	)
	// create an option set
	for _, lo := range l.listingProto.Item.Options {
		optSet[lo.Name] = struct{}{}
	}
	// walk through purchase options and verify
	for _, po := range itemOptions {
		// that they are available on the listing
		if _, ok := optSet[po.Name]; !ok {
			return fmt.Errorf("unknown item option (%s)", po.Name)
		}
		// that they haven't already been applied
		if _, ok := checkedOpt[po.Name]; ok {
			return fmt.Errorf("item option (%s) applied more than once", po.Name)
		}
		checkedOpt[po.Name] = struct{}{}
	}
	return nil
}

func ValidateListingOptions(listingItemOptions []*pb.Listing_Item_Option, itemOptions []option) ([]*pb.Order_Item_Option, error) {
	var validatedListingOptions []*pb.Order_Item_Option
	listingOptions := make(map[string]*pb.Listing_Item_Option)
	for _, opt := range listingItemOptions {
		listingOptions[strings.ToLower(opt.Name)] = opt
	}
	for _, uopt := range itemOptions {
		_, ok := listingOptions[strings.ToLower(uopt.Name)]
		if !ok {
			return nil, errors.New("selected variant not in listing")
		}
		delete(listingOptions, strings.ToLower(uopt.Name))
	}
	if len(listingOptions) > 0 {
		return nil, errors.New("Not all options were selected")
	}

	for _, option := range itemOptions {
		o := &pb.Order_Item_Option{
			Name:  option.Name,
			Value: option.Value,
		}
		validatedListingOptions = append(validatedListingOptions, o)
	}
	return validatedListingOptions, nil
}
