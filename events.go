// events.go
package main

import (
	// Pastikan database/sql digunakan
	"encoding/json"
	"net/http"
)

// Event struct untuk merepresentasikan event dalam database
type Event struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Time   string `json:"time"`
	Detail string `json:"detail"`
}

// CreateEventHandler untuk membuat event baru
func createEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Gunakan koneksi database yang sudah ada di main.go
	db := setupDatabase()
	defer db.Close()

	// Menyimpan event ke database
	result, err := db.Exec("INSERT INTO events (event_name, event_time, event_detail) VALUES (?, ?, ?)", event.Name, event.Time, event.Detail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ambil ID event yang baru dibuat
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to retrieve last inserted ID", http.StatusInternalServerError)
		return
	}

	// Update ID event dengan ID yang dihasilkan
	event.ID = int(lastInsertID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// GetEventsHandler untuk mengambil semua event
func getEventsHandler(w http.ResponseWriter, r *http.Request) {
	db := setupDatabase()
	defer db.Close()

	rows, err := db.Query("SELECT id, event_name, event_time, event_detail FROM events")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Name, &event.Time, &event.Detail); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		events = append(events, event)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// DeleteEventHandler untuk menghapus event berdasarkan ID
func deleteEventHandler(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("id")
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	// Hapus event dari database berdasarkan ID
	_, err := db.Exec("DELETE FROM events WHERE id = ?", eventID)
	if err != nil {
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event deleted successfuly"))
}

// UpdateEventHandler untuk memperbarui event berdasarkan ID
func updateEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventID := r.URL.Query().Get("id")
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	// Update event di database berdasarkan ID
	_, err := db.Exec("UPDATE events SET event_name = ?, event_time = ?, event_detail = ? WHERE id = ?", event.Name, event.Time, event.Detail, eventID)
	if err != nil {
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(event)
}
