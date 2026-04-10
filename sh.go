package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/argon2"
)

const usage = `Usage: 
  sh                              - Lock data from MySQL
  sh unlock <CID> <SALT_HEX>      - Unlock data from IPFS
  sh image <PATH>                 - Lock image file
  sh image-unlock <CID> <SALT> <OUTPUT> - Unlock image to output dir
  sh video <PATH>                 - Lock video file
  sh video-unlock <CID> <SALT> <OUTPUT> - Unlock video to output dir
`

const (
	saltSize   = 16
	timeCost   = 3
	memoryCost = 1 * 1024
	threads    = 1
	keyLength  = 32
)

type AtomicResult struct {
	CID      string
	TxHash   string
	DataHash string
	KeyHash  string
}

type EncryptedData struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	DataHash   string `json:"data_hash"`
	AAD        string `json:"aad"`
}

type AnchorRecord struct {
	CID       string `json:"cid"`
	Timestamp int64  `json:"timestamp"`
	DataHash  string `json:"data_hash"`
	KeyHash   string `json:"key_hash"`
}

type BesuClient struct {
	client  *ethclient.Client
	privKey *ecdsa.PrivateKey
	from    common.Address
	chainID *big.Int
}

type IPFSResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

type Tbl struct {
	ID              int       `json:"id"`
	Emr_No          int       `json:"emr_no"`
	Pelayanan_id    int       `json:"pelayanan_id"`
	Waktu           time.Time `json:"waktu"`
	Heart_rate      int       `json:"heart_rate"`
	Respirasi       int       `json:"respirasi"`
	Jarak_kasur_cm  int       `json:"jarak_kasur_cm"`
	Glukosa         int       `json:"glukosa"`
	Berat_badan_kg  float64   `json:"berat_badan_kg"`
	Sistolik        int       `json:"sistolik"`
	Diastolik       int       `json:"diastolik"`
	Fall_detected   int       `json:"fall_detected"`
	Tinggi_badan_cm int       `json:"tingi_badan_cm"`
	Bmi             float64   `json:"bmi"`
	Kolesterol      int       `json:"kolestrol"`
	Asam_urat       float64   `json:"asam_urat"`
	Suhu            float64   `json:"suhu"`
	Spo2            int       `json:"spo2"`
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("ndak ada .env cuy")
	}
}

func GenerateSalt() []byte {
	salt := make([]byte, saltSize)
	rand.Read(salt)
	return salt
}

func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, timeCost, memoryCost, threads, keyLength)
}

func DeriveKeyFromEnv() ([]byte, []byte, error) {
	password := os.Getenv("ENCRYPTION_PASSWORD")
	if password == "" {
		return nil, nil, fmt.Errorf("ENCRYPTION_PASSWORD not set")
	}
	salt := GenerateSalt()
	key := DeriveKey(password, salt)
	return key, salt, nil
}

func deriveKeyFromSaved(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, timeCost, memoryCost, threads, keyLength)
}

func HashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func GetBesuRPCURL() string {
	url := os.Getenv("BESU_RPC_URL")
	if url == "" {
		url = "http://192.168.1.132:8545"
	}
	return url
}

func GetDBDSN() string {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root123@tcp(192.168.1.239:3306)/darsinurse?parseTime=true"
	}
	return dsn
}

func NewBesuClient(rpcURL, privateKeyHex string) (*BesuClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}

	pubKey := privKey.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("publicKey is not of type *ecdsa.PublicKey")
	}
	from := crypto.PubkeyToAddress(*pubKeyECDSA)
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	return &BesuClient{
		client:  client,
		privKey: privKey,
		from:    from,
		chainID: chainID,
	}, nil
}

func (b *BesuClient) AnchorCID(cid string, dataHash string, keyHash string) (string, error) {
	anchorData := AnchorRecord{
		CID:       cid,
		Timestamp: time.Now().Unix(),
		DataHash:  dataHash,
		KeyHash:   keyHash,
	}

	dataBytes, err := json.Marshal(anchorData)
	if err != nil {
		return "", err
	}

	nonce, err := b.client.PendingNonceAt(context.Background(), b.from)
	if err != nil {
		return "", err
	}

	gasPrice, err := b.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(
		nonce,
		b.from,
		big.NewInt(0),
		300000,
		gasPrice,
		dataBytes,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(b.chainID), b.privKey)
	if err != nil {
		return "", err
	}

	err = b.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

func (b *BesuClient) GetTransactionReceipt(txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	return b.client.TransactionReceipt(context.Background(), hash)
}

func UploadToIPFS(data []byte) (string, error) {
	ipfsAPI := "http://192.168.1.132:5001/api/v0/add"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "encrypted.json")
	if err != nil {
		return "", err
	}
	part.Write(data)
	writer.Close()

	req, err := http.NewRequest("POST", ipfsAPI, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result IPFSResponse
	json.Unmarshal(body, &result)
	return result.Hash, nil
}

func RetrieveFromIPFS(cid string) (*EncryptedData, error) {
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

	var encrypted EncryptedData
	err = json.Unmarshal(body, &encrypted)
	if err != nil {
		return nil, err
	}

	return &encrypted, nil
}

func GetIPFSURL(hash string) string {
	return "http://192.168.1.132:8080/ipfs/" + hash
}

func EncryptAES256GCM(plaintext []byte, key []byte, aad []byte) (*EncryptedData, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	dataHash := HashData(plaintext)
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)

	return &EncryptedData{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		DataHash:   dataHash,
		AAD:        base64.StdEncoding.EncodeToString(aad),
	}, nil
}

func DecryptAES256GCM(encrypted *EncryptedData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, err
	}

	aad, err := base64.StdEncoding.DecodeString(encrypted.AAD)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, err
	}

	if HashData(plaintext) != encrypted.DataHash {
		return nil, fmt.Errorf("integrity check failed")
	}

	return plaintext, nil
}

func AtomicFlow(besu *BesuClient, data []byte, aad []byte, key []byte) (*AtomicResult, error) {
	dataHash := HashData(data)

	encrypted, err := EncryptAES256GCM(data, key, aad)
	if err != nil {
		return nil, fmt.Errorf("encrypt error: %w", err)
	}

	encryptedJSON, _ := json.Marshal(encrypted)
	cid, err := UploadToIPFS(encryptedJSON)
	if err != nil {
		return nil, fmt.Errorf("ipfs upload error: %w", err)
	}

	keyHash := HashData(key)
	txHash, err := besu.AnchorCID(cid, dataHash, keyHash)
	if err != nil {
		return nil, fmt.Errorf("besu anchor error: %w", err)
	}

	return &AtomicResult{
		CID:      cid,
		TxHash:   txHash,
		DataHash: dataHash,
		KeyHash:  keyHash,
	}, nil
}

func Unlock(cid string, saltHex string) ([]Tbl, error) {
	password := os.Getenv("ENCRYPTION_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("ENCRYPTION_PASSWORD not set")
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}

	key := deriveKeyFromSaved(password, salt)

	encrypted, err := RetrieveFromIPFS(cid)
	if err != nil {
		return nil, fmt.Errorf("retrieve error: %w", err)
	}

	plaintext, err := DecryptAES256GCM(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt error: %w", err)
	}

	var data []Tbl
	err = json.Unmarshal(plaintext, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return data, nil
}

func runLock() {
	loadEnv()

	db, err := sql.Open("mysql", GetDBDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	key, salt, err := DeriveKeyFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Key derived (salt: %x)\n", salt)

	besu, err := NewBesuClient(GetBesuRPCURL(), os.Getenv("BESU_PRIVATE_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("SELECT * FROM `vitals-experiment`")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var data []Tbl
	for rows.Next() {
		var tbl Tbl
		if err := rows.Scan(
			&tbl.ID,
			&tbl.Emr_No,
			&tbl.Pelayanan_id,
			&tbl.Waktu,
			&tbl.Heart_rate,
			&tbl.Respirasi,
			&tbl.Jarak_kasur_cm,
			&tbl.Glukosa,
			&tbl.Berat_badan_kg,
			&tbl.Sistolik,
			&tbl.Diastolik,
			&tbl.Fall_detected,
			&tbl.Tinggi_badan_cm,
			&tbl.Bmi,
			&tbl.Kolesterol,
			&tbl.Asam_urat,
			&tbl.Suhu,
			&tbl.Spo2,
		); err != nil {
			log.Fatal(err)
		}
		data = append(data, tbl)
	}

	selectedIndices := []int{1}
	var filtered []Tbl
	for _, i := range selectedIndices {
		if i < len(data) {
			filtered = append(filtered, data[i])
		}
	}

	aad := []byte(fmt.Sprintf("medical-record-%d-%d", filtered[0].ID, time.Now().Unix()))

	filteredJSON, err := json.Marshal(filtered)
	if err != nil {
		log.Fatal(err)
	}

	result, err := AtomicFlow(besu, filteredJSON, aad, key)
	if err != nil {
		log.Fatal(err)
	}

	sqlite, err := InitSQLite("./benchmark_results.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sqlite.Close()

	record := &MultiProtocolRecord{
		SourceTable: "vitals-experiment",
		SourceID:    int(filtered[0].ID),
		Salt:        hex.EncodeToString(salt),
		DataHash:    result.DataHash,
		KeyHash:     result.KeyHash,
		IPFSCID:     result.CID,
		BesuTxHash:  result.TxHash,
		CreatedAt:   time.Now(),
	}

	id, err := sqlite.InsertRecord(record)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("SQLite ID: %d\n", id)
}

func runUnlock(cid string, saltHex string) {
	loadEnv()

	data, err := Unlock(cid, saltHex)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== UNLOCK SUCCESS ===")
	for _, d := range data {
		fmt.Printf("ID: %d, EMR: %d, Waktu: %s\n", d.ID, d.Emr_No, d.Waktu.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Heart Rate: %d, Suhu: %.1f, SpO2: %d\n", d.Heart_rate, d.Suhu, d.Spo2)
		fmt.Printf("  Berat: %.2f kg, BMI: %.2f\n", d.Berat_badan_kg, d.Bmi)
		fmt.Println("---")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "unlock":
		if len(os.Args) < 4 {
			fmt.Println(usage)
			os.Exit(1)
		}
		cid := os.Args[2]
		saltHex := os.Args[3]
		runUnlock(cid, saltHex)
	case "image":
		if len(os.Args) < 3 {
			fmt.Println(usage)
			os.Exit(1)
		}
		imagePath := os.Args[2]
		RunLockImage(imagePath)
	case "image-unlock":
		if len(os.Args) < 5 {
			fmt.Println(usage)
			os.Exit(1)
		}
		cid := os.Args[2]
		saltHex := os.Args[3]
		outputPath := os.Args[4]
		RunUnlockImage(cid, saltHex, outputPath)
	case "video":
		if len(os.Args) < 3 {
			fmt.Println(usage)
			os.Exit(1)
		}
		videoPath := os.Args[2]
		RunLockVideo(videoPath)
	case "video-unlock":
		if len(os.Args) < 5 {
			fmt.Println(usage)
			os.Exit(1)
		}
		cid := os.Args[2]
		saltHex := os.Args[3]
		outputPath := os.Args[4]
		RunUnlockVideo(cid, saltHex, outputPath)
	default:
		runLock()
	}
}
