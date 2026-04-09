package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"time"
)

type MultiProtocolRecord struct {
	ID          int64
	SourceTable string
	SourceID    int
	Salt        string
	DataHash    string
	KeyHash     string

	IPFSCID     string
	ArweaveTx   string
	FilecoinCID string

	BesuTxHash string

	EncryptionTimeMs int
	IPFSTimeMs       int
	ArweaveTimeMs    int
	FilecoinTimeMs   int
	BesuTimeMs       int
	ConcurrentTimeMs int
	TotalTimeMs      int

	CreatedAt time.Time
}

type SQLiteDB struct {
	db *sql.DB
}

func InitSQLite(dbPath string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS image_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	file_name TEXT NOT NULL,
	file_type TEXT NOT NULL,
	file_size INTEGER,
	source_path TEXT,
	salt TEXT NOT NULL,
	data_hash TEXT NOT NULL,
	key_hash TEXT NOT NULL,
	ipfs_cid TEXT,
	arweave_tx TEXT,
	filecoin_cid TEXT,
	besu_tx_hash TEXT,
	encryption_time_ms INTEGER,
	ipfs_time_ms INTEGER,
	arweave_time_ms INTEGER,
	filecoin_time_ms INTEGER,
	besu_time_ms INTEGER,
	concurrent_time_ms INTEGER,
	total_time_ms INTEGER,
	metadata TEXT
);
CREATE INDEX IF NOT EXISTS idx_image_created ON image_records(created_at);
CREATE INDEX IF NOT EXISTS idx_image_ipfs ON image_records(ipfs_cid);

	CREATE TABLE IF NOT EXISTS multi_protocol_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, 
		source_table TEXT DEFAULT 'virtals-experiment',
		source_id INTEGER NOT NULL,
		salt TEXT NOT NULL,
		data_hash TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		ipfs_cid TEXT,
		arweave_tx TEXT,
		filecoin_cid TEXT,
		besu_tx_hash TEXT,
		encryption_time_ms INTEGER,
		ipfs_time_ms INTEGER,
		arweave_time_ms INTEGER,
		filecoin_time_ms INTEGER,
		besu_time_ms INTEGER,
		concurrent_time_ms INTEGER,
		total_time_ms INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_created_at ON multi_protocol_records(created_at);
	CREATE INDEX IF NOT EXISTS idx_source ON multi_protocol_records(source_id);
	CREATE INDEX IF NOT EXISTS idx_ipfs ON multi_protocol_records(ipfs_cid);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return &SQLiteDB{db: db}, nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

func (s *SQLiteDB) InsertRecord(record *MultiProtocolRecord) (int64, error) {
	query := `
	INSERT INTO multi_protocol_records (
		source_table, source_id, salt, data_hash, key_hash,
		ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
		encryption_time_ms, ipfs_time_ms, arweave_time_ms,
		filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		record.SourceTable, record.SourceID, record.Salt, record.DataHash, record.KeyHash,
		record.IPFSCID, record.ArweaveTx, record.FilecoinCID, record.BesuTxHash,
		record.EncryptionTimeMs, record.IPFSTimeMs, record.ArweaveTimeMs,
		record.FilecoinTimeMs, record.BesuTimeMs, record.ConcurrentTimeMs, record.TotalTimeMs,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *SQLiteDB) GetRecordByID(id int64) (*MultiProtocolRecord, error) {
	query := `
	SELECT id, source_table, source_id, salt, data_hash, key_hash,
	ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
	encryption_time_ms, ipfs_time_ms, arweave_time_ms,
	filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms,
	created_at FROM multi_protocol_records WHERE id = ?
	`

	record := &MultiProtocolRecord{}
	err := s.db.QueryRow(query, id).Scan(
		&record.ID, &record.SourceTable, &record.SourceID, &record.Salt,
		&record.DataHash, &record.KeyHash, &record.IPFSCID, &record.ArweaveTx,
		&record.FilecoinCID, &record.BesuTxHash, &record.EncryptionTimeMs,
		&record.IPFSTimeMs, &record.ArweaveTimeMs, &record.FilecoinTimeMs,
		&record.BesuTimeMs, &record.ConcurrentTimeMs, &record.TotalTimeMs,
		&record.CreatedAt,
	)

	return record, err
}

func (s *SQLiteDB) GetAllRecords() ([]MultiProtocolRecord, error) {
	query := `
	SELECT id, source_table, source_id, salt, data_hash, key_hash,
		ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
		encryption_time_ms, ipfs_time_ms, arweave_time_ms,
		filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms, created_at
	FROM multi_protocol_records ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []MultiProtocolRecord
	for rows.Next() {
		var r MultiProtocolRecord
		err := rows.Scan(
			&r.ID, &r.SourceTable, &r.SourceID, &r.Salt, &r.DataHash, &r.KeyHash,
			&r.IPFSCID, &r.ArweaveTx, &r.FilecoinCID, &r.BesuTxHash,
			&r.EncryptionTimeMs, &r.IPFSTimeMs, &r.ArweaveTimeMs,
			&r.FilecoinTimeMs, &r.BesuTimeMs, &r.ConcurrentTimeMs, &r.TotalTimeMs,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func (s *SQLiteDB) GetRecordsByDateRange(start, end time.Time) ([]MultiProtocolRecord, error) {
	query := `
	SELECT id, source_table, source_id, salt, data_hash, key_hash,
		ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
		encryption_time_ms, ipfs_time_ms, arweave_time_ms,
		filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms, created_at
	FROM multi_protocol_records 
	WHERE created_at BETWEEN ? AND ?
	ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []MultiProtocolRecord
	for rows.Next() {
		var r MultiProtocolRecord
		err := rows.Scan(
			&r.ID, &r.SourceTable, &r.SourceID, &r.Salt, &r.DataHash, &r.KeyHash,
			&r.IPFSCID, &r.ArweaveTx, &r.FilecoinCID, &r.BesuTxHash,
			&r.EncryptionTimeMs, &r.IPFSTimeMs, &r.ArweaveTimeMs,
			&r.FilecoinTimeMs, &r.BesuTimeMs, &r.ConcurrentTimeMs, &r.TotalTimeMs,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
func (s *SQLiteDB) ExportToCSV(filename string) error {
	records, err := s.GetAllRecords()
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	fmt.Fprintln(file, "id,created_at,source_id,salt,data_hash,key_hash, ipfs_cid,arweave_tx,filecoin_cid,besu_tx_hash,encryption_ms, ipfs_ms,arweave_ms,filecoin_ms,besu_ms,concurrent_ms,total_ms")

	for _, r := range records {
		fmt.Fprintf(file, "%d,%s,%d,%s,%s,%s,%s,%s,%s,%s,%d,%d,%d,%d,%d,%d,%d\n",
			r.ID, r.CreatedAt.Format("2006-01-02 15:04:05"), r.SourceID,
			r.Salt, r.DataHash, r.KeyHash,
			r.IPFSCID, r.ArweaveTx, r.FilecoinCID, r.BesuTxHash,
			r.EncryptionTimeMs, r.IPFSTimeMs, r.ArweaveTimeMs,
			r.FilecoinTimeMs, r.BesuTimeMs, r.ConcurrentTimeMs, r.TotalTimeMs,
		)
	}
	return nil
}
