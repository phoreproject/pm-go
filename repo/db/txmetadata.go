package db

import (
	"database/sql"
	"sync"

	"github.com/phoreproject/pm-go/repo"
)

type TxMetadataDB struct {
	modelStore
}

func NewTransactionMetadataStore(db *sql.DB, lock *sync.Mutex) repo.TransactionMetadataStore {
	return &TxMetadataDB{modelStore{db, lock}}
}

func (t *TxMetadataDB) Put(m repo.Metadata) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	stmt, err := t.db.Prepare("insert or replace into txmetadata(txid, address, memo, orderID, thumbnail, canBumpFee) values(?,?,?,?,?,?)")
	if err != nil {
		log.Errorf("prepring txmetadata sql for order (%s): %s", m.OrderID, err.Error())
		return err
	}
	defer stmt.Close()
	bumpable := 0
	if m.CanBumpFee {
		bumpable = 1
	}
	_, err = stmt.Exec(m.Txid, m.Address, m.Memo, m.OrderID, m.Thumbnail, bumpable)
	if err != nil {
		log.Errorf("putting txmetadata for order (%s): %s", m.OrderID, err.Error())
		return err
	}
	return nil
}

func (t *TxMetadataDB) Get(txid string) (repo.Metadata, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	var m repo.Metadata
	stmt, err := t.db.Prepare("select txid, address, memo, orderID, thumbnail, canBumpFee from txmetadata where txid=?")
	if err != nil {
		return m, err
	}
	defer stmt.Close()
	var id, address, memo, orderId, thumbnail string
	var canBumpFee int
	err = stmt.QueryRow(txid).Scan(&id, &address, &memo, &orderId, &thumbnail, &canBumpFee)
	if err != nil {
		return m, err
	}
	bumpable := false
	if canBumpFee > 0 {
		bumpable = true
	}
	m = repo.Metadata{Txid: id, Address: address, Memo: memo, OrderID: orderId, Thumbnail: thumbnail, CanBumpFee: bumpable}
	return m, nil
}

func (t *TxMetadataDB) GetAll() (map[string]repo.Metadata, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	ret := make(map[string]repo.Metadata)
	stm := "select txid, address, memo, orderID, thumbnail, canBumpFee from txmetadata"
	rows, err := t.db.Query(stm)
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	for rows.Next() {
		var txid, address, memo, orderId, thumbnail string
		var canBumpFee int
		if err := rows.Scan(&txid, &address, &memo, &orderId, &thumbnail, &canBumpFee); err != nil {
			return ret, err
		}
		bumpable := false
		if canBumpFee > 0 {
			bumpable = true
		}
		m := repo.Metadata{
			Txid:       txid,
			Address:    address,
			Memo:       memo,
			OrderID:    orderId,
			Thumbnail:  thumbnail,
			CanBumpFee: bumpable,
		}
		ret[txid] = m
	}
	return ret, nil
}

func (t *TxMetadataDB) Delete(txid string) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	_, err := t.db.Exec("delete from txmetadata where txid=?", txid)
	if err != nil {
		return err
	}
	return nil
}
