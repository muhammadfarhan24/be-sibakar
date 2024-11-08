package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// Booking struct
type Booking struct {
	Namalengkap  string `json:"namalengkap"`
	Nama_divisi  string `json:"nama_divisi"`
	SelectedSeat string `json:"selected_seat"`
	Status       string `json:"status"`
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
	fmt.Printf("Booking data received: %+v\n", booking)

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
	// Memasukkan data pemesanan
	result, err := db.Exec(`
		INSERT INTO bookings (selected_seat) 
		VALUES (?)`, booking.SelectedSeat)
	if err != nil {
		return fmt.Errorf("failed to insert into bookings table: %v", err)
	}

	bookingID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	// // Tentukan status berdasarkan status dari frontend
	// var status string
	// if booking.Status == "occupied" {
	// 	status = "occupied" // Menggunakan nilai ENUM
	// } else {
	// 	status = "tersedia" // Menggunakan nilai ENUM
	// }

	// Memasukkan data pemesanan dan status ke logactivity
	// Memasukkan data pemesanan ke dalam tabel logactivity
	_, err = db.Exec(`
		INSERT INTO logactivity (id, namalengkap, nama_divisi, selected_seat, status)
		VALUES (?, ?, ?, ?, ?)`,
		bookingID, booking.Namalengkap, booking.Nama_divisi, booking.SelectedSeat, booking.Status)
	if err != nil {
		return fmt.Errorf("failed to insert into logactivity table: %v", err)
	}

	return nil
}

func getOccupiedSeatsHandler(w http.ResponseWriter, r *http.Request) {
	// Setup database connection
	db := setupDatabase()
	defer db.Close()

	// Query to get occupied seats
	rows, err := db.Query(`
		SELECT selected_seat
		FROM logactivity
		WHERE status = 'occupied'`) // Pastikan menggunakan status 'occupied'
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect the occupied seats
	var occupiedSeats []string
	for rows.Next() {
		var seat string
		if err := rows.Scan(&seat); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		occupiedSeats = append(occupiedSeats, seat)
	}

	// Debug log: Print the occupied seats
	fmt.Printf("Occupied seats: %+v\n", occupiedSeats)

	// Ensure that the response is an array in JSON format
	w.Header().Set("Content-Type", "application/json")
	if len(occupiedSeats) == 0 {
		// Return an empty array if no occupied seats
		json.NewEncoder(w).Encode([]string{})
		return
	}
	json.NewEncoder(w).Encode(occupiedSeats)
}

func getBookingActivityHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil ID booking dari parameter query
	bookingID := r.URL.Query().Get("booking_id")
	if bookingID == "" {
		http.Error(w, "Booking ID is required", http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	// Ambil aktivitas terkait pemesanan ini dari tabel logactivity
	rows, err := db.Query(`
		SELECT namalengkap, nama_divisi, selected_seat, status, created_at 
		FROM logactivity 
		WHERE id = ?`, bookingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var activities []map[string]interface{}
	for rows.Next() {
		var namalengkap, namaDivisi, selectedSeat, status, createdAt string
		if err := rows.Scan(&namalengkap, &namaDivisi, &selectedSeat, &status, &createdAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		activities = append(activities, map[string]interface{}{
			"namalengkap":   namalengkap,
			"nama_divisi":   namaDivisi,
			"selected_seat": selectedSeat,
			"status":        status,
			"created_at":    createdAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}