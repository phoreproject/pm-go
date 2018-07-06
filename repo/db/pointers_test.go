package db_test

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/OpenBazaar/openbazaar-go/ipfs"
	"github.com/phoreproject/openbazaar-go/repo"
	"github.com/phoreproject/openbazaar-go/repo/db"
	"github.com/phoreproject/openbazaar-go/schema"
	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	ps "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	multihash "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func mustNewPointer() ipfs.Pointer {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	h, err := multihash.Encode(randBytes, multihash.SHA2_256)
	if err != nil {
		panic(err)
	}
	id, err := peer.IDFromBytes(h)
	if err != nil {
		panic(err)
	}
	maAddr, err := ma.NewMultiaddr("/ipfs/QmamudHQGtztShX7Nc9HcczehdpGGWpFBWu2JvKWcpELxr/")
	if err != nil {
		panic(err)
	}
	k, err := cid.Decode("QmamudHQGtztShX7Nc9HcczehdpGGWpFBWu2JvKWcpELxr")
	if err != nil {
		panic(err)
	}
	cancelID, err := peer.IDB58Decode("QmbwSMS35CaYKdrYBvvR9aHU9FzeWhjJ7E3jLKeR2DWrs3")
	if err != nil {
		panic(err)
	}
	return ipfs.Pointer{
		k,
		ps.PeerInfo{
			ID:    id,
			Addrs: []ma.Multiaddr{maAddr},
		},
		ipfs.MESSAGE,
		time.Now(),
		&cancelID,
	}
}

func buildNewPointerStore() (repo.PointerStore, func(), error) {
	appSchema := schema.MustNewCustomSchemaManager(schema.SchemaContext{
		DataPath:        schema.GenerateTempPath(),
		TestModeEnabled: true,
	})
	if err := appSchema.BuildSchemaDirectories(); err != nil {
		return nil, nil, err
	}
	if err := appSchema.InitializeDatabase(); err != nil {
		return nil, nil, err
	}
	database, err := appSchema.OpenDatabase()
	if err != nil {
		return nil, nil, err
	}
	return db.NewPointerStore(database, new(sync.Mutex)), appSchema.DestroySchemaDirectories, nil
}

func TestPointersPut(t *testing.T) {
	pdb, teardown, err := buildNewPointerStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	pointer := mustNewPointer()
	err = pdb.Put(pointer)
	if err != nil {
		t.Error(err)
	}

	stmt, _ := pdb.PrepareQuery("select pointerID, key, address, cancelID, purpose, timestamp from pointers where pointerID=?")
	defer stmt.Close()

	var pointerID string
	var key string
	var address string
	var purpose int
	var timestamp int
	var cancelID string
	err = stmt.QueryRow(pointer.Value.ID.Pretty()).Scan(&pointerID, &key, &address, &cancelID, &purpose, &timestamp)
	if err != nil {
		t.Error(err)
	}
	if pointerID != pointer.Value.ID.Pretty() || timestamp <= 0 || key != pointer.Cid.String() || purpose != 1 || cancelID != pointer.CancelID.Pretty() {
		t.Error("Pointer returned incorrect values")
	}
	err = pdb.Put(pointer)
	if err == nil {
		t.Error("Allowed duplicate pointer")
	}
}

func TestDeletePointer(t *testing.T) {
	pdb, teardown, err := buildNewPointerStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	pointer := mustNewPointer()
	pdb.Put(pointer)
	err = pdb.Delete(pointer.Value.ID)
	if err != nil {
		t.Error("Pointer delete failed")
	}
	stmt, _ := pdb.PrepareQuery("select pointerID from pointers where pointerID=?")
	defer stmt.Close()

	var pointerID string
	err = stmt.QueryRow(pointer.Value.ID.Pretty()).Scan(&pointerID)
	if err == nil {
		t.Error("Pointer delete failed")
	}
}

func TestDeleteAllPointers(t *testing.T) {
	pdb, teardown, err := buildNewPointerStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	p := mustNewPointer()
	p.Purpose = ipfs.MODERATOR
	pdb.Put(p)
	err = pdb.DeleteAll(ipfs.MODERATOR)
	if err != nil {
		t.Error("Pointer delete failed")
	}
	stmt, _ := pdb.PrepareQuery("select pointerID from pointers where purpose=?")
	defer stmt.Close()

	var pointerID string
	err = stmt.QueryRow(ipfs.MODERATOR).Scan(&pointerID)
	if err == nil {
		t.Error("Pointer delete all failed")
	}
}

func TestGetAllPointers(t *testing.T) {
	pdb, teardown, err := buildNewPointerStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	pointer := mustNewPointer()
	pdb.Put(pointer)
	pointers, err := pdb.GetAll()
	if err != nil {
		t.Error("Get all pointers returned error")
	}
	for _, p := range pointers {
		if p.Purpose != pointer.Purpose {
			t.Error("Get all pointers returned incorrect data")
		}
		if p.Value.ID != pointer.Value.ID {
			t.Error("Get all pointers returned incorrect data")
		}
		if !p.Cid.Equals(pointer.Cid) {
			t.Error("Get all pointers returned incorrect data")
		}
		if p.CancelID.Pretty() != pointer.CancelID.Pretty() {
			t.Error("Get all pointers returned incorrect data")
		}
	}
}

func TestPointersDB_GetByPurpose(t *testing.T) {
	pdb, teardown, err := buildNewPointerStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	pdb.Put(mustNewPointer())
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	h, _ := multihash.Encode(randBytes, multihash.SHA2_256)
	id, _ := peer.IDFromBytes(h)
	maAddr, _ := ma.NewMultiaddr("/ipfs/QmamudHQGtztShX7Nc9HcczehdpGGWpFBWu2JvKWcpELxr/")
	k, _ := cid.Decode("QmamudHQGtztShX7Nc9HcczehdpGGWpFBWu2JvKWcpELxr")
	m := ipfs.Pointer{
		k,
		ps.PeerInfo{
			ID:    id,
			Addrs: []ma.Multiaddr{maAddr},
		},
		ipfs.MODERATOR,
		time.Now(),
		nil,
	}
	err = pdb.Put(m)
	pointers, err := pdb.GetByPurpose(ipfs.MODERATOR)
	if err != nil {
		t.Error("Get pointers returned error")
	}
	if len(pointers) != 1 {
		t.Error("Returned incorrect number of pointers")
	}
	for _, p := range pointers {
		if p.Purpose != m.Purpose {
			t.Error("Get pointers returned incorrect data")
		}
		if p.Value.ID != m.Value.ID {
			t.Error("Get pointers returned incorrect data")
		}
		if !p.Cid.Equals(m.Cid) {
			t.Error("Get pointers returned incorrect data")
		}
		if p.CancelID != nil {
			t.Error("Get pointers returned incorrect data")
		}
	}
}

func TestPointersDB_Get(t *testing.T) {
	pdb, teardown, err := buildNewPointerStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	pdb.Put(mustNewPointer())
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	h, _ := multihash.Encode(randBytes, multihash.SHA2_256)
	id, _ := peer.IDFromBytes(h)
	maAddr, _ := ma.NewMultiaddr("/ipfs/QmamudHQGtztShX7Nc9HcczehdpGGWpFBWu2JvKWcpELxr/")
	k, _ := cid.Decode("QmamudHQGtztShX7Nc9HcczehdpGGWpFBWu2JvKWcpELxr")
	m := ipfs.Pointer{
		k,
		ps.PeerInfo{
			ID:    id,
			Addrs: []ma.Multiaddr{maAddr},
		},
		ipfs.MODERATOR,
		time.Now(),
		nil,
	}
	err = pdb.Put(m)
	p, err := pdb.Get(id)
	if err != nil {
		t.Error("Get pointers returned error")
	}

	if p.Purpose != m.Purpose {
		t.Error("Get pointers returned incorrect data")
	}
	if p.Value.ID != m.Value.ID {
		t.Error("Get pointers returned incorrect data")
	}
	if !p.Cid.Equals(m.Cid) {
		t.Error("Get pointers returned incorrect data")
	}
	if p.CancelID != nil {
		t.Error("Get pointers returned incorrect data")
	}
}
