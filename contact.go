package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// Struktur data untuk Contact
type Contact struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Message   string `json:"message"`
}

// Fungsi untuk menyimpan data kontak ke dalam database
func saveContact(db *sql.DB, contact Contact) error {
	_, err := db.Exec(`
		INSERT INTO contacts (first_name, last_name, email, phone, message) 
		VALUES (?, ?, ?, ?, ?)`,
		contact.FirstName, contact.LastName, contact.Email, contact.Phone, contact.Message)
	return err
}
func getContacts(db *sql.DB) ([]Contact, error) {
	rows, err := db.Query("SELECT first_name, last_name, email, phone, message FROM contacts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var contact Contact
		if err := rows.Scan(&contact.FirstName, &contact.LastName, &contact.Email, &contact.Phone, &contact.Message); err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	return contacts, nil
}

// Handler untuk menangani form kontak
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Mendekode data kontak yang dikirimkan dalam request body
	var contact Contact
	if err := json.NewDecoder(r.Body).Decode(&contact); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Setup koneksi database
	db := setupDatabase()
	defer db.Close()

	// Simpan data kontak ke dalam database
	if err := saveContact(db, contact); err != nil {
		http.Error(w, "Failed to save contact", http.StatusInternalServerError)
		return
	}

	// Tampilkan log di konsol server untuk debugging
	fmt.Printf("New contact form submission: %+v\n", contact)

	// Kirim response dengan pesan terima kasih
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Thank you for contacting us! We will get back to you shortly.",
	})
}

func getContactsHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := setupDatabase()
	defer db.Close()

	contacts, err := getContacts(db)
	if err != nil {
		http.Error(w, "failed to retrieve contacts", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}
