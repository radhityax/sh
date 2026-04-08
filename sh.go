package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type AnchorRecord struct {
	CID       string `json:"cid"`
	Timestamp int64  `json:"timestamp"`
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

type EncryptedData struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("ndak ada .env cuy")
	}
}

func GetPrivateKey() string {
	return os.Getenv("BESU_PRIVATE_KEY")
}

func GetBesuRPCURL() string {
	url := os.Getenv("BESU_RPC_URL")
	if url == "" {
		url = "http://192.168.1.132:8545"
	}
	return url
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

func (b *BesuClient) AnchorCID(cid string, keyHash string) (string, error) {
	anchorData := AnchorRecord{
		CID:       cid,
		Timestamp: time.Now().Unix(),
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
		30000,
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

func GetIPFSURL(hash string) string {
	return "http://192.168.1.132:8080/ipfs/" + hash
}
func GenerateKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func EncryptAES256GCM(plaintext []byte, key []byte) (*EncryptedData, error) {
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

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
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

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func main() {
	loadEnv()

	db, err := sql.Open("mysql", "root:root123@tcp(192.168.1.239:3306)/darsinurse?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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

	key := GenerateKey()
	jsonData, err := json.Marshal(filtered)
	if err != nil {
		log.Fatal(err)
	}

	encrypted, err := EncryptAES256GCM(jsonData, key)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== ENCRYPTED DATA ===")
	encryptedJSON, _ := json.MarshalIndent(encrypted, "", "  ")

	cid, err := UploadToIPFS(encryptedJSON)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(encryptedJSON))

	decrypted, err := DecryptAES256GCM(encrypted, key)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n=== DECRYPTED DATA ===")
	var decryptedData []Tbl
	json.Unmarshal(decrypted, &decryptedData)
	decryptedJSON, _ := json.MarshalIndent(decryptedData, "", "  ")
	fmt.Println(string(decryptedJSON))

	fmt.Println("IPFS CID:", cid)
	fmt.Println("IPFS URL:", GetIPFSURL(cid))

	keyString := base64.StdEncoding.EncodeToString(key)
	fmt.Println("Key:", keyString)

	privateKey := GetPrivateKey()
	rpcURL := GetBesuRPCURL()

	besu, err := NewBesuClient(rpcURL, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	keyHash := crypto.Keccak256Hash([]byte(keyString)).Hex()
	txHash, err := besu.AnchorCID(cid, keyHash)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== HYPERLEDGER BESU ANCHORING ===")
	fmt.Println("Transaction Hash:", txHash)
	fmt.Println("CID:", cid)
	fmt.Println("Key Hash:", keyHash)
}
