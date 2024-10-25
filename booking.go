package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// Booking struct
type Booking struct {
	Namalengkap  string `json:"namalengkap"`
	Namadivisi   string `json:"namadivisi"`
	SelectedSeat string `json:"selectedSeat"`
}

// Booking handler
func bookingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var booking Booking
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	if err := saveBooking(db, booking); err != nil {
		http.Error(w, "Error saving booking", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}

func saveBooking(db *sql.DB, booking Booking) error {
	_, err := db.Exec("INSERT INTO bookings (namalengkap, namadivisi, selected_seat) VALUES (?, ?, ?)",
		booking.Namalengkap, booking.Namadivisi, booking.SelectedSeat)
	return err
}
