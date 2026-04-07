package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	_ "strings"
	"time"
)

type Interval int

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

func main() {

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

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonData))
}
