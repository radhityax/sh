package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ImageRecord struct {
	ID         int64
	FileName   string
	FileType   string /* "jpg", "png" */
	FileSize   int64
	SourcePath string

	Salt     string
	DataHash string
	KeyHash  string

	IPFSCID     string
	ArweaveTx   string
	FilecoinCID string

	BesuTxHash       string
	Metadata         string
	EncryptionTimeMs int
	IPFSTimeMs       int
	ArweaveTimeMs    int
	FilecoinTimeMs   int
	BesuTimeMs       int
	ConcurrentTimeMs int
	TotalTimeMs      int
	CreatedAt        time.Time
}

type EncryptedImage struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	DataHash   string `json:"data_hash"`
	AAD        string `json:"aad"`
	FileName   string `json:"file_name"`
	FileType   string `json:"file_type"`
	FileSize   int64  `json:"file_size"`
	Metadata   string `json:"metadata"`
}

func ReadImageFile(filePath string) ([]byte, string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, "", 0, err
	}

	fileSize := info.Size()

	ext := filepath.Ext(filePath)
	fileType := strings.TrimPrefix(ext, ".")
	if fileType == "jpeg" {
		fileType = "jpg"
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, "", 0, err
	}

	return content, fileType, fileSize, nil
}

func EncryptImage(imageData []byte, fileName string, fileType string, fileSize int64, key []byte, aad []byte) (*EncryptedImage, error) {
	encrypted, err := EncryptAES256GCM(imageData, key, aad)
	if err != nil {
		return nil, err
	}

	encryptedImage := &EncryptedImage{
		Nonce:      encrypted.Nonce,
		Ciphertext: encrypted.Ciphertext,
		DataHash:   encrypted.DataHash,
		AAD:        encrypted.AAD,
		FileName:   fileName,
		FileType:   fileType,
		FileSize:   fileSize,
		Metadata:   fmt.Sprintf("%s|%s|%d", fileName, fileType, fileSize),
	}

	return encryptedImage, nil
}

func DecryptImage(encryptedImage *EncryptedImage, key []byte) ([]byte, error) {
	decryptedData := &EncryptedData{
		Nonce:      encryptedImage.Nonce,
		Ciphertext: encryptedImage.Ciphertext,
		DataHash:   encryptedImage.DataHash,
		AAD:        encryptedImage.AAD,
	}
	return DecryptAES256GCM(decryptedData, key)
}

func (s *SQLiteDB) InsertImageRecord(record *ImageRecord) (int64, error) {
	query := `
	INSERT INTO image_records (
		file_name, file_type, file_size, source_path,
		salt, data_hash, key_hash,
		ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
		encryption_time_ms, ipfs_time_ms, arweave_time_ms,
		filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms,
		metadata
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(query,
		record.FileName, record.FileType, record.FileSize, record.SourcePath,
		record.Salt, record.DataHash, record.KeyHash,
		record.IPFSCID, record.ArweaveTx, record.FilecoinCID, record.BesuTxHash,
		record.EncryptionTimeMs, record.IPFSTimeMs, record.ArweaveTimeMs,
		record.FilecoinTimeMs, record.BesuTimeMs, record.ConcurrentTimeMs, record.TotalTimeMs,
		record.Metadata,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
func (s *SQLiteDB) GetImageRecordByID(id int64) (*ImageRecord, error) {
	query := `
	SELECT id, file_name, file_type, file_size, source_path,
		salt, data_hash, key_hash,
		ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
		encryption_time_ms, ipfs_time_ms, arweave_time_ms,
		filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms,
		metadata, created_at
	FROM image_records WHERE id = ?
	`
	record := &ImageRecord{}
	err := s.db.QueryRow(query, id).Scan(
		&record.ID, &record.FileName, &record.FileType, &record.FileSize, &record.SourcePath,
		&record.Salt, &record.DataHash, &record.KeyHash,
		&record.IPFSCID, &record.ArweaveTx, &record.FilecoinCID, &record.BesuTxHash,
		&record.EncryptionTimeMs, &record.IPFSTimeMs, &record.ArweaveTimeMs,
		&record.FilecoinTimeMs, &record.BesuTimeMs, &record.ConcurrentTimeMs, &record.TotalTimeMs,
		&record.Metadata, &record.CreatedAt,
	)
	return record, err
}
func (s *SQLiteDB) GetAllImageRecords() ([]ImageRecord, error) {
	query := `
	SELECT id, file_name, file_type, file_size, source_path,
		salt, data_hash, key_hash,
		ipfs_cid, arweave_tx, filecoin_cid, besu_tx_hash,
		encryption_time_ms, ipfs_time_ms, arweave_time_ms,
		filecoin_time_ms, besu_time_ms, concurrent_time_ms, total_time_ms,
		metadata, created_at
	FROM image_records ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []ImageRecord
	for rows.Next() {
		var r ImageRecord
		err := rows.Scan(
			&r.ID, &r.FileName, &r.FileType, &r.FileSize, &r.SourcePath,
			&r.Salt, &r.DataHash, &r.KeyHash,
			&r.IPFSCID, &r.ArweaveTx, &r.FilecoinCID, &r.BesuTxHash,
			&r.EncryptionTimeMs, &r.IPFSTimeMs, &r.ArweaveTimeMs,
			&r.FilecoinTimeMs, &r.BesuTimeMs, &r.ConcurrentTimeMs, &r.TotalTimeMs,
			&r.Metadata, &r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
func ProcessImageFile(imagePath string, besu *BesuClient, sqlite *SQLiteDB) (*ImageRecord, error) {
	startTime := time.Now()
	imageData, fileType, fileSize, err := ReadImageFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("read image error: %w", err)
	}
	fileName := filepath.Base(imagePath)
	key, salt, err := DeriveKeyFromEnv()
	if err != nil {
		return nil, fmt.Errorf("key derivation error: %w", err)
	}
	aad := []byte(fmt.Sprintf("image-%s-%d", fileName, time.Now().Unix()))
	encryptStart := time.Now()
	encryptedImage, err := EncryptImage(imageData, fileName, fileType, fileSize, key, aad)
	if err != nil {
		return nil, fmt.Errorf("encrypt error: %w", err)
	}
	encryptTime := time.Since(encryptStart).Milliseconds()
	encryptedJSON, _ := json.Marshal(encryptedImage)
	encodedData := base64.StdEncoding.EncodeToString(encryptedJSON)
	dataHash := HashData(imageData)
	ipfsStart := time.Now()
	ipfsCID, err := UploadToIPFS([]byte(encodedData))
	if err != nil {
		return nil, fmt.Errorf("ipfs upload error: %w", err)
	}
	ipfsTime := time.Since(ipfsStart).Milliseconds()
	arweaveTime := 0
	filecoinTime := 0
	keyHash := HashData(key)
	anchorStart := time.Now()
	txHash, err := besu.AnchorCID(ipfsCID, dataHash, keyHash)
	if err != nil {
		return nil, fmt.Errorf("besu anchor error: %w", err)
	}
	besuTime := time.Since(anchorStart).Milliseconds()
	totalTime := time.Since(startTime).Milliseconds()
	record := &ImageRecord{
		FileName:         fileName,
		FileType:         fileType,
		FileSize:         fileSize,
		SourcePath:       imagePath,
		Salt:             hex.EncodeToString(salt),
		DataHash:         dataHash,
		KeyHash:          keyHash,
		IPFSCID:          ipfsCID,
		BesuTxHash:       txHash,
		Metadata:         encryptedImage.Metadata,
		EncryptionTimeMs: int(encryptTime),
		IPFSTimeMs:       int(ipfsTime),
		ArweaveTimeMs:    int(arweaveTime),
		FilecoinTimeMs:   int(filecoinTime),
		BesuTimeMs:       int(besuTime),
		TotalTimeMs:      int(totalTime),
		CreatedAt:        time.Now(),
	}
	id, err := sqlite.InsertImageRecord(record)
	if err != nil {
		return nil, fmt.Errorf("sqlite error: %w", err)
	}
	record.ID = id
	return record, nil
}
func RetrieveRawFromIPFS(cid string) ([]byte, error) {
	url := "http://192.168.1.132:8080/ipfs/" + cid

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func RetrieveImageFromIPFS(cid string, key []byte, outputPath string) (*ImageRecord, error) {
	rawContent, err := RetrieveRawFromIPFS(cid)
	if err != nil {
		return nil, fmt.Errorf("retrieve error: %w", err)
	}

	encryptedJSON, err := base64.StdEncoding.DecodeString(string(rawContent))
	if err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	var encryptedImage EncryptedImage
	err = json.Unmarshal(encryptedJSON, &encryptedImage)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	imageData, err := DecryptImage(&encryptedImage, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt error: %w", err)
	}

	outputFileName := filepath.Join(outputPath, encryptedImage.FileName)
	err = os.WriteFile(outputFileName, imageData, 0644)
	if err != nil {
		return nil, fmt.Errorf("write error: %w", err)
	}

	return &ImageRecord{
		FileName: encryptedImage.FileName,
		FileType: encryptedImage.FileType,
		FileSize: encryptedImage.FileSize,
	}, nil
}
func RunLockImage(imagePath string) {
	loadEnv()
	sqlite, err := InitSQLite("./image_results.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sqlite.Close()
	besu, err := NewBesuClient(GetBesuRPCURL(), os.Getenv("BESU_PRIVATE_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	record, err := ProcessImageFile(imagePath, besu, sqlite)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Lock <3")
	fmt.Printf("File: %s\n", record.FileName)
	fmt.Printf("Type: %s\n", record.FileType)
	fmt.Printf("Size: %d bytes\n", record.FileSize)
	fmt.Printf("Salt: %s\n", record.Salt)
	fmt.Printf("IPFS CID: %s\n", record.IPFSCID)
	fmt.Printf("Besu Tx: %s\n", record.BesuTxHash)
	fmt.Printf("Data Hash: %s\n", record.DataHash)
	fmt.Printf("Encryption: %d ms\n", record.EncryptionTimeMs)
	fmt.Printf("IPFS Upload: %d ms\n", record.IPFSTimeMs)
	fmt.Printf("Besu Anchor: %d ms\n", record.BesuTimeMs)
	fmt.Printf("Total: %d ms\n", record.TotalTimeMs)
}
func RunUnlockImage(cid string, saltHex string, outputPath string) {
	loadEnv()
	password := os.Getenv("ENCRYPTION_PASSWORD")
	salt, _ := hex.DecodeString(saltHex)
	key := deriveKeyFromSaved(password, salt)
	_, err := RetrieveImageFromIPFS(cid, key, outputPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Unlock <3")
	fmt.Printf("Output: %s\n", outputPath)
}
