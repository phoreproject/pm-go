package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/phoreproject/pm-go/core"
	"github.com/phoreproject/pm-go/pb"
	"github.com/phoreproject/pm-go/repo"
	"github.com/phoreproject/pm-go/test"
	"github.com/phoreproject/pm-go/test/factory"
)

func TestMain(m *testing.M) {
	// Create a test server
	gateway, err := newTestGateway()
	if err != nil {
		log.Fatal(err)
	}
	defer gateway.Close()

	go func() {
		err = gateway.Serve(true)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Run tests
	retCode := m.Run()

	os.Exit(retCode)
}

func TestSettings(t *testing.T) {
	// Create, Read, Update, Patch
	runAPITests(t, apiTests{
		{"POST", "/ob/settings", settingsJSON, 200, settingsJSON},
		{"GET", "/ob/settings", "", 200, settingsJSON},
		{"POST", "/ob/settings", settingsJSON, 409, settingsAlreadyExistsJSON},
		{"PUT", "/ob/settings", settingsUpdateJSON, 200, "{}"},
		{"GET", "/ob/settings", "", 200, settingsUpdateJSON},
		{"PUT", "/ob/settings", settingsUpdateJSON, 200, "{}"},
		{"GET", "/ob/settings", "", 200, settingsUpdateJSON},
		{"PATCH", "/ob/settings", settingsPatchJSON, 200, "{}"},
		{"GET", "/ob/settings", "", 200, settingsPatchedJSON},
	})

	// Invalid JSON
	runAPITests(t, apiTests{
		{"POST", "/ob/settings", settingsMalformedJSON, 400, settingsMalformedJSONResponse},
	})

	// Invalid JSON
	runAPITests(t, apiTests{
		{"POST", "/ob/settings", settingsJSON, 200, settingsJSON},
		{"GET", "/ob/settings", "", 200, settingsJSON},
		{"PUT", "/ob/settings", settingsMalformedJSON, 400, settingsMalformedJSONResponse},
	})
}

func TestProfile(t *testing.T) {
	// Create, Update
	runAPITests(t, apiTests{
		{"POST", "/ob/profile", profileJSON, 200, anyResponseJSON},
		{"POST", "/ob/profile", profileJSON, 409, AlreadyExistsUsePUTJSON("Profile")},
		{"PUT", "/ob/profile", profileUpdateJSON, 200, anyResponseJSON},
		{"PUT", "/ob/profile", profileUpdatedJSON, 200, anyResponseJSON},
	})
}

func TestAvatar(t *testing.T) {
	// Setting an avatar fails if we don't have a profile
	runAPITests(t, apiTests{
		{"POST", "/ob/avatar", avatarValidJSON, 500, anyResponseJSON},
	})

	// It succeeds if we have a profile and the image data is valid
	runAPITests(t, apiTests{
		{"POST", "/ob/profile", profileJSON, 200, anyResponseJSON},
		{"POST", "/ob/avatar", avatarValidJSON, 200, avatarValidJSONResponse},
	})

	// Test invalid image data
	runAPITests(t, apiTests{
		{"POST", "/ob/profile", profileJSON, 200, anyResponseJSON},
		{"POST", "/ob/avatar", avatarUnexpectedEOFJSON, 500, avatarUnexpectedEOFJSONResponse},
	})

	runAPITests(t, apiTests{
		{"POST", "/ob/profile", profileJSON, 200, anyResponseJSON},
		{"POST", "/ob/avatar", avatarInvalidTQJSON, 500, avatarInvalidTQJSONResponse},
	})
}

func TestImages(t *testing.T) {
	// Valid image
	runAPITests(t, apiTests{
		{"POST", "/ob/images", imageValidJSON, 200, imageValidJSONResponse},
	})
}

func TestHeader(t *testing.T) {
	// Setting an header fails if we don't have a profile
	runAPITests(t, apiTests{
		{"POST", "/ob/header", headerValidJSON, 500, anyResponseJSON},
	})

	// It succeeds if we have a profile and the image data is valid
	runAPITests(t, apiTests{
		{"POST", "/ob/profile", profileJSON, 200, anyResponseJSON},
		{"POST", "/ob/header", headerValidJSON, 200, headerValidJSONResponse},
	})
}

func TestModerator(t *testing.T) {
	// Fails without profile
	runAPITests(t, apiTests{
		{"PUT", "/ob/moderator", moderatorValidJSON, http.StatusConflict, anyResponseJSON},
	})

	// Works with profile
	runAPITests(t, apiTests{
		{"POST", "/ob/profile", profileJSON, 200, anyResponseJSON},

		// TODO: Enable after fixing bug that requires peers in order to set moderator status
		// {"PUT", "/ob/moderator", moderatorValidJSON, 200, `{}`},

		// // Update
		// {"PUT", "/ob/moderator", moderatorUpdatedValidJSON, 200, `{}`},
		{"DELETE", "/ob/moderator", "", 200, `{}`},
	})
}

func TestMessageSignVerify(t *testing.T) {
	const (
		signMessageJSON = `{
	"content": "test"
}`
		verifyMessageJSON = `{
	"content": "test",
	"signature": "fac9dec1ce872c931bda1af85f9107e8733b42ed6401bc989a84b6b53ad263290d9bd9d470f046024884f502ecb7af50de2fea11268e82dcb1c72d50753c330a",
	"pubkey": "080112203f94c7707af68ede9ddd24a16edd813146550df565eda8fb81114476ccfe6b78",
	"peerId": "QmRmisSghsxUMrTQZ5vmqFroxxuCXJqXwXoTc21q5cefmM"
}`
	)

	runAPITests(t, apiTests{
		{"POST", "/ob/signmessage", signMessageJSON, 200, anyResponseJSON},
		{"POST", "/ob/verifymessage", verifyMessageJSON, 200, anyResponseJSON},
	})
}

func TestMessageSignsURLChars(t *testing.T) {
	const (
		validateSignWorksWithURLChars = `{
		"content":"QmdQBWA75xQSMZpTibQ2G83enNdriz2v14tetGvNrpr5KB/this-is-a-social-post"
	}`
	)

	runAPITests(t, apiTests{
		{"POST", "/ob/signmessage", validateSignWorksWithURLChars, 200, anyResponseJSON},
	})
}

func TestListingsAcceptedCurrencies(t *testing.T) {
	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, factory.NewListing("ron-swanson-tshirt")), 200, anyResponseJSON},
	})

	req, err := buildRequest("GET", "/ob/listings", "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := testHTTPClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure correct status code
	if resp.StatusCode != 200 {
		t.Fatalf("Wanted status 200, got %d", resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	var respObj []struct {
		AcceptedCurrencies []string `json:"acceptedCurrencies"`
	}
	err = json.Unmarshal(respBody, &respObj)
	if err != nil {
		t.Fatal(err)
	}

	if len(respObj) != 1 {
		t.Fatal("Listings should contain exactly 1 listing")
	}

	if respObj[0].AcceptedCurrencies == nil {
		t.Fatal("Listing should contain exactly 1 acceptedCurrency")
	}

	if respObj[0].AcceptedCurrencies[0] != "TBTC" {
		t.Fatal("Listing acceptedCurrencies should contain 'TBTC'")
	}
}

func TestListingAcceptedCurrencies(t *testing.T) {
	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, factory.NewListing("ron-swanson-tshirt")), 200, anyResponseJSON},
	})

	req, err := buildRequest("GET", "/ob/listing/ron-swanson-tshirt", "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := testHTTPClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure correct status code
	if resp.StatusCode != 200 {
		t.Fatalf("Wanted status 200, got %d", resp.StatusCode)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	respObj := struct {
		Listing struct {
			Metadata struct {
				AcceptedCurrencies []string `json:"acceptedCurrencies"`
			} `json:"metadata"`
		} `json:"listing"`
	}{}
	err = json.Unmarshal(respBody, &respObj)
	if err != nil {
		t.Fatal(err)
	}

	if len(respObj.Listing.Metadata.AcceptedCurrencies) != 1 {
		t.Fatal("Listing should contain exactly 1 acceptedCurrency")
	}

	if respObj.Listing.Metadata.AcceptedCurrencies[0] != "TBTC" {
		t.Fatal("Listing acceptedCurrenc9es should contain 'TBTC'")
	}
}

func TestListings(t *testing.T) {
	goodListingJSON := jsonFor(t, factory.NewListing("ron-swanson-tshirt"))
	updatedListing := factory.NewListing("ron-swanson-tshirt")
	updatedListing.Taxes = []*pb.Listing_Tax{
		{
			Percentage:  17,
			TaxShipping: true,
			TaxType:     "Sales tax",
			TaxRegions:  []pb.CountryCode{pb.CountryCode_UNITED_STATES},
		},
	}
	updatedListingJSON := jsonFor(t, updatedListing)

	runAPITests(t, apiTests{
		{"GET", "/ob/listings", "", 200, `[]`},
		{"GET", "/ob/inventory", "", 200, `{}`},

		// Invalid creates
		{"POST", "/ob/listing", `{`, 400, jsonUnexpectedEOF},

		{"GET", "/ob/listings", "", 200, `[]`},
		{"GET", "/ob/inventory", "", 200, `{}`},

		// TODO: Add support for improved JSON matching to since contracts
		// change each test run due to signatures

		// Create/Get
		{"GET", "/ob/listing/ron-swanson-tshirt", "", 404, NotFoundJSON("Listing")},
		{"POST", "/ob/listing", goodListingJSON, 200, `{"slug": "ron-swanson-tshirt"}`},
		{"GET", "/ob/listing/ron-swanson-tshirt", "", 200, anyResponseJSON},
		{"POST", "/ob/listing", updatedListingJSON, 409, AlreadyExistsUsePUTJSON("Listing")},

		// TODO: Add support for improved JSON matching to since contracts
		// change each test run due to signatures
		{"GET", "/ob/listings", "", 200, anyResponseJSON},

		// TODO: This returns `inventoryJSONResponse` but slices are unordered
		// so they don't get considered equal. Figure out a way to fix that.
		{"GET", "/ob/inventory", "", 200, anyResponseJSON},

		// Update inventory
		{"POST", "/ob/inventory", inventoryUpdateJSON, 200, `{}`},

		// Update/Get Listing
		{"PUT", "/ob/listing", updatedListingJSON, 200, `{}`},
		{"GET", "/ob/listing/ron-swanson-tshirt", "", 200, anyResponseJSON},

		// Delete/Get
		{"DELETE", "/ob/listing/ron-swanson-tshirt", "", 200, `{}`},
		{"DELETE", "/ob/listing/ron-swanson-tshirt", "", 404, NotFoundJSON("Listing")},
		{"GET", "/ob/listing/ron-swanson-tshirt", "", 404, NotFoundJSON("Listing")},

		// Mutate non-existing listings
		{"PUT", "/ob/listing", updatedListingJSON, 404, NotFoundJSON("Listing")},
		{"DELETE", "/ob/listing/ron-swanson-tshirt", "", 404, NotFoundJSON("Listing")},

		// Bulk update currency in listings
		{"POST", "/ob/bulkupdatecurrency", bulkUpdateCurrencyJSON, 200, `{"success": "true"}`},
	})
}

func TestCryptoListings(t *testing.T) {
	listing := factory.NewCryptoListing("crypto")
	updatedListing := *listing

	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`},
		{"GET", "/ob/listing/crypto", jsonFor(t, &updatedListing), 200, anyResponseJSON},

		{"PUT", "/ob/listing", jsonFor(t, &updatedListing), 200, "{}"},
		{"PUT", "/ob/listing", jsonFor(t, &updatedListing), 200, "{}"},
		{"GET", "/ob/listing/crypto", jsonFor(t, &updatedListing), 200, anyResponseJSON},

		{"DELETE", "/ob/listing/crypto", "", 200, `{}`},
		{"DELETE", "/ob/listing/crypto", "", 404, NotFoundJSON("Listing")},
		{"GET", "/ob/listing/crypto", "", 404, NotFoundJSON("Listing")},
	})
}

func TestCryptoListingsPriceModifier(t *testing.T) {
	outOfRangeErr := core.ErrPriceModifierOutOfRange{
		Min: core.PriceModifierMin,
		Max: core.PriceModifierMax,
	}

	listing := factory.NewCryptoListing("crypto")
	listing.Metadata.PriceModifier = core.PriceModifierMax
	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`},
		{"GET", "/ob/listing/crypto", jsonFor(t, listing), 200, anyResponseJSON},
	})

	listing.Metadata.PriceModifier = core.PriceModifierMax + 0.001
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`,
	})

	listing.Metadata.PriceModifier = core.PriceModifierMax + 0.01
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(outOfRangeErr),
	})

	listing.Metadata.PriceModifier = core.PriceModifierMin - 0.001
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`,
	})

	listing.Metadata.PriceModifier = core.PriceModifierMin - 1
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(outOfRangeErr),
	})

	listing.Item.Price = 1
	var f core.ErrMarketPriceListingIllegalField = "item.price"
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(f),
	})

	listing.Metadata.Format = pb.Listing_Metadata_FIXED_PRICE
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`,
	})
}

func TestListingsQuantity(t *testing.T) {
	listing := factory.NewListing("crypto")
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`,
	})

	listing.Item.Skus[0].Quantity = 0
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, anyResponseJSON,
	})

	listing.Item.Skus[0].Quantity = -1
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, anyResponseJSON,
	})
}

func TestCryptoListingsQuantity(t *testing.T) {
	listing := factory.NewCryptoListing("crypto")
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 200, `{"slug": "crypto"}`,
	})

	listing.Item.Skus[0].Quantity = 0
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(core.ErrCryptocurrencySkuQuantityInvalid),
	})

	listing.Item.Skus[0].Quantity = -1
	runAPITest(t, apiTest{
		"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(core.ErrCryptocurrencySkuQuantityInvalid),
	})
}

func TestCryptoListingsNoCoinType(t *testing.T) {
	listing := factory.NewCryptoListing("crypto")
	listing.Metadata.CoinType = ""

	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(core.ErrCryptocurrencyListingCoinTypeRequired)},
	})
}

func TestCryptoListingsCoinDivisibilityIncorrect(t *testing.T) {
	listing := factory.NewCryptoListing("crypto")
	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 200, anyResponseJSON},
	})

	listing.Metadata.CoinDivisibility = 1e7
	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(core.ErrListingCoinDivisibilityIncorrect)},
	})

	listing.Metadata.CoinDivisibility = 0
	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(core.ErrListingCoinDivisibilityIncorrect)},
	})
}

func TestCryptoListingsIllegalFields(t *testing.T) {
	runTest := func(listing *pb.Listing, err error) {
		runAPITests(t, apiTests{
			{"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(err)},
		})
	}

	physicalListing := factory.NewListing("physical")

	listing := factory.NewCryptoListing("crypto")
	listing.Metadata.PricingCurrency = "btc"
	runTest(listing, core.ErrCryptocurrencyListingIllegalField("metadata.pricingCurrency"))

	listing = factory.NewCryptoListing("crypto")
	listing.Item.Condition = "new"
	runTest(listing, core.ErrCryptocurrencyListingIllegalField("item.condition"))

	listing = factory.NewCryptoListing("crypto")
	listing.Item.Options = physicalListing.Item.Options
	runTest(listing, core.ErrCryptocurrencyListingIllegalField("item.options"))

	listing = factory.NewCryptoListing("crypto")
	listing.ShippingOptions = physicalListing.ShippingOptions
	runTest(listing, core.ErrCryptocurrencyListingIllegalField("shippingOptions"))

	listing = factory.NewCryptoListing("crypto")
	listing.Coupons = physicalListing.Coupons
	runTest(listing, core.ErrCryptocurrencyListingIllegalField("coupons"))
}

func TestMarketRatePrice(t *testing.T) {
	listing := factory.NewListing("listing")
	listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
	listing.Item.Price = 1

	runAPITests(t, apiTests{
		{"POST", "/ob/listing", jsonFor(t, listing), 500, errorResponseJSON(core.ErrMarketPriceListingIllegalField("item.price"))},
	})
}

func TestStatus(t *testing.T) {
	runAPITests(t, apiTests{
		{"GET", "/ob/status", "", 400, anyResponseJSON},
		{"GET", "/ob/status/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", "", 200, anyResponseJSON},
	})
}

func TestWallet(t *testing.T) {
	runAPITests(t, apiTests{
		{"GET", "/wallet/address", "", 200, walletAddressJSONResponse},
		{"GET", "/wallet/balance", "", 200, walletBalanceJSONResponse},
		{"GET", "/wallet/mnemonic", "", 200, walletMneumonicJSONResponse},
		{"POST", "/wallet/spend", spendJSON, 400, insuffientFundsJSON},
		// TODO: Test successful spend on regnet with coins
	})
}

func TestExchangeRates(t *testing.T) {
	runAPITests(t, apiTests{
		{"GET", "/ob/exchangerates", "", 500, invalidCoinJSON},
		{"GET", "/ob/exchangerates/", "", 500, invalidCoinJSON},
		{"GET", "/ob/exchangerates/BTC", "", 200, anyResponseJSON},
	})
}

func TestConfig(t *testing.T) {
	runAPITests(t, apiTests{
		// TODO: Need better JSON matching
		{"GET", "/ob/config", "", 200, anyResponseJSON},
	})
}

func Test404(t *testing.T) {
	// Test undefined endpoints
	runAPITests(t, apiTests{
		{"GET", "/ob/a", "{}", 404, notFoundJSON},
		{"PUT", "/ob/a", "{}", 404, notFoundJSON},
		{"POST", "/ob/a", "{}", 404, notFoundJSON},
		{"PATCH", "/ob/a", "{}", 404, notFoundJSON},
		{"DELETE", "/ob/a", "{}", 404, notFoundJSON},
	})
}

func TestPosts(t *testing.T) {
	runAPITests(t, apiTests{
		{"GET", "/ob/posts", "", 200, `[]`},

		// Invalid creates
		{"POST", "/ob/post", `{`, 400, jsonUnexpectedEOF},

		{"GET", "/ob/posts", "", 200, `[]`},

		// Create/Get
		{"GET", "/ob/post/test1", "", 404, NotFoundJSON("Post")},
		{"POST", "/ob/post", postJSON, 200, postJSONResponse},
		{"GET", "/ob/post/test1", "", 200, anyResponseJSON},
		{"POST", "/ob/post", postUpdateJSON, 409, AlreadyExistsUsePUTJSON("Post")},

		{"GET", "/ob/posts", "", 200, anyResponseJSON},

		// Update/Get Post
		{"PUT", "/ob/post", postUpdateJSON, 200, `{}`},
		{"GET", "/ob/post/test1", "", 200, anyResponseJSON},

		// Delete/Get
		{"DELETE", "/ob/post/test1", "", 200, `{}`},
		{"DELETE", "/ob/post/test1", "", 404, NotFoundJSON("Post")},
		{"GET", "/ob/post/test1", "", 404, NotFoundJSON("Post")},

		// Mutate non-existing listings
		{"PUT", "/ob/post", postUpdateJSON, 404, NotFoundJSON("Post")},
		{"DELETE", "/ob/post/test1", "", 404, NotFoundJSON("Post")},
	})
}

func TestCloseDisputeBlocksWhenExpired(t *testing.T) {
	dbSetup := func(testRepo *test.Repository) error {
		paymentCoin := repo.CurrencyCode("BTC")
		expired := factory.NewExpiredDisputeCaseRecord()
		expired.CaseID = "expiredCase"
		expired.PaymentCoin = &paymentCoin
		for _, r := range []*repo.DisputeCaseRecord{expired} {
			if err := testRepo.DB.Cases().PutRecord(r); err != nil {
				return err
			}
			if err := testRepo.DB.Cases().UpdateBuyerInfo(r.CaseID, r.BuyerContract, []string{}, r.BuyerPayoutAddress, r.BuyerOutpoints); err != nil {
				return err
			}
		}
		return nil
	}
	expiredPostJSON := `{"orderId":"expiredCase","resolution":"","buyerPercentage":100.0,"vendorPercentage":0.0}`
	runAPITestsWithSetup(t, apiTests{
		{"POST", "/ob/closedispute", expiredPostJSON, 400, anyResponseJSON},
	}, dbSetup, nil)
}

func TestZECSalesCannotReleaseEscrow(t *testing.T) {
	sale := factory.NewSaleRecord()
	sale.Contract.VendorListings[0].Metadata.AcceptedCurrencies = []string{"ZEC"}
	sale.Contract.BuyerOrder.Payment.Coin = "ZEC"
	dbSetup := func(testRepo *test.Repository) error {
		if err := testRepo.DB.Sales().Put(sale.OrderID, *sale.Contract, sale.OrderState, false); err != nil {
			return err
		}
		return nil
	}
	runAPITestsWithSetup(t, apiTests{
		{"POST", "/ob/releaseescrow", fmt.Sprintf(`{"orderId":"%s"}`, sale.OrderID), 400, anyResponseJSON},
	}, dbSetup, nil)
}

func TestSalesGet(t *testing.T) {
	sale := factory.NewSaleRecord()
	sale.Contract.VendorListings[0].Metadata.AcceptedCurrencies = []string{"BTC"}
	sale.Contract.VendorListings[0].Metadata.CoinType = "ZEC"
	sale.Contract.VendorListings[0].Metadata.ContractType = pb.Listing_Metadata_CRYPTOCURRENCY
	dbSetup := func(testRepo *test.Repository) error {
		return testRepo.DB.Sales().Put(sale.OrderID, *sale.Contract, sale.OrderState, false)
	}

	runAPITestsWithSetup(t, apiTests{
		{"GET", "/ob/sales", "", 200, anyResponseJSON},
	}, dbSetup, nil)

	respBytes, err := httpGet("/ob/sales")
	if err != nil {
		t.Fatal(err)
	}
	respObj := struct {
		Sales []repo.Sale `json:"sales"`
	}{}
	err = json.Unmarshal(respBytes, &respObj)
	if err != nil {
		t.Fatal(err)
	}

	actualSale := respObj.Sales[0]

	if actualSale.BuyerHandle != sale.Contract.BuyerOrder.BuyerID.Handle {
		t.Fatal("Incorrect buyerHandle:", actualSale.BuyerHandle, "\nwanted:", sale.Contract.BuyerOrder.BuyerID.Handle)
	}
	if actualSale.BuyerId != sale.Contract.BuyerOrder.BuyerID.PeerID {
		t.Fatal("Incorrect buyerId:", actualSale.BuyerId, "\nwanted:", sale.Contract.BuyerOrder.BuyerID.PeerID)
	}
	if actualSale.CoinType != sale.Contract.VendorListings[0].Metadata.CoinType {
		t.Fatal("Incorrect coinType:", actualSale.CoinType, "\nwanted:", sale.Contract.VendorListings[0].Metadata.CoinType)
	}
	if actualSale.OrderId != sale.OrderID {
		t.Fatal("Incorrect orderId:", actualSale.OrderId, "\nwanted:", sale.OrderID)
	}
	if actualSale.PaymentCoin != sale.Contract.VendorListings[0].Metadata.AcceptedCurrencies[0] {
		t.Fatal("Incorrect paymentCoin:", actualSale.PaymentCoin, "\nwanted:", sale.Contract.VendorListings[0].Metadata.AcceptedCurrencies[0])
	}
	if actualSale.ShippingAddress != sale.Contract.BuyerOrder.Shipping.Address {
		t.Fatal("Incorrect shippingAddress:", actualSale.ShippingAddress, "\nwanted:", sale.Contract.BuyerOrder.Shipping.Address)
	}
	if actualSale.ShippingName != sale.Contract.BuyerOrder.Shipping.ShipTo {
		t.Fatal("Incorrect shippingName:", actualSale.ShippingName, "\nwanted:", sale.Contract.BuyerOrder.Shipping.ShipTo)
	}
	if actualSale.State != sale.OrderState.String() {
		t.Fatal("Incorrect state:", actualSale.State, "\nwanted:", sale.OrderState.String())
	}
}
func TestPurchasesGet(t *testing.T) {
	purchase := factory.NewPurchaseRecord()
	purchase.Contract.VendorListings[0].Metadata.AcceptedCurrencies = []string{"BTC"}
	purchase.Contract.VendorListings[0].Metadata.CoinType = "ZEC"
	purchase.Contract.VendorListings[0].Metadata.ContractType = pb.Listing_Metadata_CRYPTOCURRENCY
	dbSetup := func(testRepo *test.Repository) error {
		return testRepo.DB.Purchases().Put(purchase.OrderID, *purchase.Contract, purchase.OrderState, false)
	}

	runAPITestsWithSetup(t, apiTests{
		{"GET", "/ob/purchases", "", 200, anyResponseJSON},
	}, dbSetup, nil)

	respBytes, err := httpGet("/ob/purchases")
	if err != nil {
		t.Fatal(err)
	}
	respObj := struct {
		Purchases []repo.Purchase `json:"purchases"`
	}{}
	err = json.Unmarshal(respBytes, &respObj)
	if err != nil {
		t.Fatal(err)
	}

	actualPurchase := respObj.Purchases[0]

	if actualPurchase.VendorHandle != purchase.Contract.VendorListings[0].VendorID.Handle {
		t.Fatal("Incorrect vendorHandle:", actualPurchase.VendorId, "\nwanted:", purchase.Contract.VendorListings[0].VendorID.Handle)
	}
	if actualPurchase.VendorId != purchase.Contract.VendorListings[0].VendorID.PeerID {
		t.Fatal("Incorrect vendorId:", actualPurchase.VendorId, "\nwanted:", purchase.Contract.VendorListings[0].VendorID.PeerID)
	}
	if actualPurchase.CoinType != purchase.Contract.VendorListings[0].Metadata.CoinType {
		t.Fatal("Incorrect coinType:", actualPurchase.CoinType, "\nwanted:", purchase.Contract.VendorListings[0].Metadata.CoinType)
	}
	if actualPurchase.OrderId != purchase.OrderID {
		t.Fatal("Incorrect orderId:", actualPurchase.OrderId, "\nwanted:", purchase.OrderID)
	}
	if actualPurchase.PaymentCoin != purchase.Contract.VendorListings[0].Metadata.AcceptedCurrencies[0] {
		t.Fatal("Incorrect paymentCoin:", actualPurchase.PaymentCoin, "\nwanted:", purchase.Contract.VendorListings[0].Metadata.AcceptedCurrencies[0])
	}
	if actualPurchase.ShippingAddress != purchase.Contract.BuyerOrder.Shipping.Address {
		t.Fatal("Incorrect shippingAddress:", actualPurchase.ShippingAddress, "\nwanted:", purchase.Contract.BuyerOrder.Shipping.Address)
	}
	if actualPurchase.ShippingName != purchase.Contract.BuyerOrder.Shipping.ShipTo {
		t.Fatal("Incorrect shippingName:", actualPurchase.ShippingName, "\nwanted:", purchase.Contract.BuyerOrder.Shipping.ShipTo)
	}
	if actualPurchase.State != purchase.OrderState.String() {
		t.Fatal("Incorrect state:", actualPurchase.State, "\nwanted:", purchase.OrderState.String())
	}
}

func TestCasesGet(t *testing.T) {
	paymentCoinCode := repo.CurrencyCode("BTC")
	disputeCaseRecord := factory.NewDisputeCaseRecord()
	disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.AcceptedCurrencies = []string{"BTC"}
	disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.CoinType = "ZEC"
	disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.ContractType = pb.Listing_Metadata_CRYPTOCURRENCY
	disputeCaseRecord.CoinType = "ZEC"
	disputeCaseRecord.PaymentCoin = &paymentCoinCode
	dbSetup := func(testRepo *test.Repository) error {
		return testRepo.DB.Cases().PutRecord(disputeCaseRecord)
	}

	runAPITestsWithSetup(t, apiTests{
		{"GET", "/ob/cases", "", 200, anyResponseJSON},
	}, dbSetup, nil)

	respBytes, err := httpGet("/ob/cases")
	if err != nil {
		t.Fatal(err)
	}
	respObj := struct {
		Cases []repo.Case `json:"cases"`
	}{}
	err = json.Unmarshal(respBytes, &respObj)
	if err != nil {
		t.Fatal(err)
	}

	actualCase := respObj.Cases[0]

	if actualCase.CoinType != disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.CoinType {
		t.Fatal("Incorrect coinType:", actualCase.CoinType, "\nwanted:", disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.CoinType)
	}
	if actualCase.PaymentCoin != disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.AcceptedCurrencies[0] {
		t.Fatal("Incorrect paymentCoin:", actualCase.PaymentCoin, "\nwanted:", disputeCaseRecord.BuyerContract.VendorListings[0].Metadata.AcceptedCurrencies[0])
	}
	if actualCase.State != disputeCaseRecord.OrderState.String() {
		t.Fatal("Incorrect state:", actualCase.State, "\nwanted:", disputeCaseRecord.OrderState.String())
	}
}

func TestNotificationsAreReturnedInExpectedOrder(t *testing.T) {
	const sameTimestampsAreReturnedInReverse = `{
    "notifications": [
        {
            "notification": {
                "notificationId": "notif3",
                "peerId": "somepeerid",
                "type": "follow"
            },
            "read": false,
	    "timestamp": "1996-07-17T23:15:45Z",
            "type": "follow"
        },
        {
            "notification": {
                "notificationId": "notif2",
                "peerId": "somepeerid",
                "type": "follow"
            },
            "read": false,
	    "timestamp": "1996-07-17T23:15:45Z",
            "type": "follow"
        },
        {
            "notification": {
                "notificationId": "notif1",
                "peerId": "somepeerid",
                "type": "follow"
            },
            "read": false,
	    "timestamp": "1996-07-17T23:15:45Z",
            "type": "follow"
        }
    ],
    "total": 3,
    "unread": 3
}`

	const sameTimestampsAreReturnedInReverseAndRespectOffsetID = `{
    "notifications": [
        {
            "notification": {
                "notificationId": "notif2",
                "peerId": "somepeerid",
                "type": "follow"
            },
            "read": false,
            "timestamp": "1996-07-17T23:15:45Z",
            "type": "follow"
        },
        {
            "notification": {
                "notificationId": "notif1",
                "peerId": "somepeerid",
                "type": "follow"
            },
            "read": false,
            "timestamp": "1996-07-17T23:15:45Z",
            "type": "follow"
        }
    ],
    "total": 0,
    "unread": 3
}`
	var (
		createdAt = time.Unix(837645345, 0)
		notif1    = &repo.Notification{
			ID:           "notif1",
			CreatedAt:    repo.NewAPITime(createdAt),
			NotifierType: repo.NotifierTypeFollowNotification,
			NotifierData: &repo.FollowNotification{
				ID:     "notif1",
				Type:   repo.NotifierTypeFollowNotification,
				PeerId: "somepeerid",
			},
		}
		notif2 = &repo.Notification{
			ID:           "notif2",
			CreatedAt:    repo.NewAPITime(createdAt),
			NotifierType: repo.NotifierTypeFollowNotification,
			NotifierData: &repo.FollowNotification{
				ID:     "notif2",
				Type:   repo.NotifierTypeFollowNotification,
				PeerId: "somepeerid",
			},
		}
		notif3 = &repo.Notification{
			ID:           "notif3",
			CreatedAt:    repo.NewAPITime(createdAt),
			NotifierType: repo.NotifierTypeFollowNotification,
			NotifierData: &repo.FollowNotification{
				ID:     "notif3",
				Type:   repo.NotifierTypeFollowNotification,
				PeerId: "somepeerid",
			},
		}
	)
	dbSetup := func(testRepo *test.Repository) error {
		for _, n := range []*repo.Notification{notif1, notif2, notif3} {
			if err := testRepo.DB.Notifications().PutRecord(n); err != nil {
				return err
			}
		}
		return nil
	}
	dbTeardown := func(testRepo *test.Repository) error {
		for _, n := range []*repo.Notification{notif1, notif2, notif3} {
			if err := testRepo.DB.Notifications().Delete(n.GetID()); err != nil {
				return err
			}
		}
		return nil
	}
	runAPITestsWithSetup(t, apiTests{
		{"GET", "/ob/notifications?limit=-1", "", 200, sameTimestampsAreReturnedInReverse},
		{"GET", "/ob/notifications?limit=-1&offsetId=notif3", "", 200, sameTimestampsAreReturnedInReverseAndRespectOffsetID},
	}, dbSetup, dbTeardown)
}

func TestResendOrderMessage(t *testing.T) {
	runAPITests(t, apiTests{
		// supports missing messageType
		{"POST", "/ob/resendordermessage", `{"orderID":"123"}`, http.StatusBadRequest, errorResponseJSON(fmt.Errorf("missing messageType argument"))},
		// supports missing order ID
		{"POST", "/ob/resendordermessage", `{"messageType":"nonexistant"}`, http.StatusBadRequest, errorResponseJSON(fmt.Errorf("missing orderID argument"))},
		// supports nonexistant message types
		{"POST", "/ob/resendordermessage", `{"orderID":"123","messageType":"nonexistant"}`, http.StatusBadRequest, errorResponseJSON(fmt.Errorf("unknown messageType (nonexistant)"))},
		// supports downcased message types, expected not to find order ID
		{"POST", "/ob/resendordermessage", `{"orderID":"123","messageType":"order"}`, http.StatusInternalServerError, errorResponseJSON(fmt.Errorf("unable to find message for order ID (123) and message type (ORDER)"))},
	})
}

// TODO: Make NewDisputeCaseRecord return a valid fixture for this valid case to work
//func TestCloseDisputeReturnsOK(t *testing.T) {
//dbSetup := func(testRepo *test.Repository) error {
//nonexpired := factory.NewDisputeCaseRecord()
//nonexpired.CaseID = "nonexpiredCase"
//for _, r := range []*repo.DisputeCaseRecord{nonexpired} {
//if err := testRepo.DB.Cases().PutRecord(r); err != nil {
//return err
//}
//if err := testRepo.DB.Cases().UpdateBuyerInfo(r.CaseID, r.BuyerContract, []string{}, r.BuyerPayoutAddress, r.BuyerOutpoints); err != nil {
//return err
//}
//}
//return nil
//}
//nonexpiredPostJSON := `{"orderId":"nonexpiredCase","resolution":"","buyerPercentage":100.0,"vendorPercentage":0.0}`
//runAPITestsWithSetup(t, apiTests{
//{"POST", "/ob/profile", moderatorProfileJSON, 200, anyResponseJSON},
//{"POST", "/ob/closedispute", nonexpiredPostJSON, 200, anyResponseJSON},
//}, dbSetup, nil)
//}
