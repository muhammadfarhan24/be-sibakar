package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"log"
	"net/http"
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

	db := setupDatabase()
	defer db.Close()

	//hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

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

	storedUser, err := getUserByUsername(db, user.Username)
	if err != nil || storedUser.Password != user.Password {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})
	tokenString, err := token.SignedString([]byte("your_secret_key"))
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}
	fmt.Println("Generated token:", tokenString)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

func getUserByUsername(db *sql.DB, username string) (User, error) {
	var user User
	err := db.QueryRow("SELECT id, username, fullname, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Fullname, &user.Password)
	return user, err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func main() {
	corsHandler := cors.Default().Handler(http.DefaultServeMux)

	// Menambahkan route
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/booking", bookingHandler)

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
