package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Setup database connection
func setupDatabase() *sql.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/sibakar" // Sesuaikan dengan pengaturan DB Anda
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

// Register user handler
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//validasi role
	if user.Role != "admin" && user.Role != "anggota" {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword) // Simpan hash ke database

	if err := registerUser(db, user); err != nil {
		http.Error(w, "Error registering user", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func registerUser(db *sql.DB, user User) error {
	_, err := db.Exec("INSERT INTO users (username, fullname, password) VALUES (?, ?, ?)", user.Username, user.Fullname, user.Password)
	return err
}

// Login user handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	// Pastikan variabel storedUser dideklarasikan di sini
	storedUser, err := getUserByUsername(db, user.Username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Debugging untuk memeriksa nilai storedUser
	fmt.Println("Stored User:", storedUser)

	// storedUser, err := getUserByUsername(db, user.Username)
	// if err != nil || !checkPasswordHash(user.Password, storedUser.Password) {
	// 	http.Error(w, "Invalid username or password", http.StatusUnauthorized)
	// 	return
	// }

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": storedUser.Username,
		"role":     storedUser.Role,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})
	tokenString, err := token.SignedString([]byte("your_secret_key"))
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": tokenString,
		"user":  storedUser,
	})
}

func getUserByUsername(db *sql.DB, username string) (User, error) {
	var user User
	err := db.QueryRow("SELECT id, username, fullname, password, role FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Fullname, &user.Password, &user.Role)
	return user, err
}

// func checkPasswordHash(password, hash string) bool {
// 	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
// 	return err == nil
// }

// Middleware untuk verifikasi token
func verifyToken(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ambil token dari header Authorization
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Token is missing", http.StatusUnauthorized)
			return
		}

		// Menghilangkan "Bearer " jika ada di depan token
		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}
		tokenString = parts[1]

		// Verifikasi token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("your_secret_key"), nil // Kunci rahasia yang sama dengan saat pembuatan token
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Token valid, lanjutkan ke handler berikutnya
		next.ServeHTTP(w, r)
	})
}

// Menambahkan Endpoint untuk Mendapatkan Data Pengguna
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	db := setupDatabase()
	defer db.Close()

	rows, err := db.Query("SELECT id, username, fullname FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Fullname); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Endpoint untuk menghapus data pengguna
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	_, err := db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Handler untuk kirim data dari tabel logactivity
func getLogActivityHandler(w http.ResponseWriter, r *http.Request) {
	db := setupDatabase()
	defer db.Close()

	rows, err := db.Query("SELECT id, namalengkap, nama_divisi, selected_seat, status FROM logactivity")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var namalengkap, namaDivisi, selectedSeat, status string
		if err := rows.Scan(&id, &namalengkap, &namaDivisi, &selectedSeat, &status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logs = append(logs, map[string]interface{}{
			"id":            id,
			"namalengkap":   namalengkap,
			"nama_divisi":   namaDivisi,
			"selected_seat": selectedSeat,
			"status":        status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// Handler untuk hapus data log
func deleteLogActivityHandler(w http.ResponseWriter, r *http.Request) {
	logID := r.URL.Query().Get("id")
	if logID == "" {
		http.Error(w, "Log activity ID is required", http.StatusBadRequest)
		return
	}

	db := setupDatabase()
	defer db.Close()

	_, err := db.Exec("DELETE FROM logactivity WHERE id = ?", logID)
	if err != nil {
		http.Error(w, "Failed to delete log activity", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Middleware untuk memverifikasi role admin
func verifyAdminRole(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Token is missing", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}
		tokenString = parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("your_secret_key"), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Verifikasi apakah role pengguna adalah "admin"
		role, ok := claims["role"].(string)
		if !ok || role != "admin" {
			http.Error(w, "Forbidden: Insufficient privileges", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	corsHandler := cors.Default().Handler(http.DefaultServeMux)

	// Menambahkan route
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/booking", bookingHandler)
	http.HandleFunc("/occupied-seats", getOccupiedSeatsHandler)
	// Penambahan route baru untuk mendapatkan data pengguna
	http.HandleFunc("/users", getUsersHandler)
	// Route untuk handler delete user
	http.HandleFunc("/users/delete", deleteUserHandler)
	// Route handler logactivity
	http.HandleFunc("/logactivity", getLogActivityHandler)
	// Route delete log
	http.HandleFunc("/logactivity/delete", deleteLogActivityHandler)
	// Endpoint yang hanya bisa diakses oleh admin
	http.HandleFunc("/admin/users", verifyAdminRole(getUsersHandler))

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
