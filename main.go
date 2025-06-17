package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func initDB() {
	var err error
	dsn := "root:root@tcp(127.0.0.1:3306)/schoolsystem"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("Error pinging database:", err)
	}
	fmt.Println("Connected to database successfully.")
}

func main() {
	initDB()
	router := gin.Default()
	router.Use(cors.Default())

	// Authentication
	router.POST("/register", registerUser)
	router.POST("/login", loginUser)

	// Student APIs
	router.GET("/students", getStudents)
	router.POST("/students", addStudent)
	router.PUT("/students/:id", updateStudent)
	router.DELETE("/students/:id", deleteStudent)

	// Marks APIs
	router.GET("/marks", getMarks)
	router.POST("/marks", addOrUpdateMark)
	router.GET("/subjects", getSubjects)

	router.Run(":9091")
}

// ==== Structs ====

type Student struct {
	ID     int    `json:"student_id"`
	Name   string `json:"student_name"`
	Age    int    `json:"Age"`
	Gender string `json:"Gender"`
	Class  int    `json:"class_id"`
}

type Subject struct {
	ID   int    `json:"subject_id"`
	Name string `json:"subject_name"`
}

type MarkEntry struct {
	StudentID     int    `json:"student_id"`
	StudentName   string `json:"student_name"`
	SubjectID     int    `json:"subject_id"`
	SubjectName   string `json:"subject_name"`
	MarksObtained int    `json:"marks_obtained"`
}

type MarkInput struct {
	StudentID     int `json:"student_id"`
	SubjectID     int `json:"subject_id"`
	MarksObtained int `json:"marks_obtained"`
}

// ==== Auth ====

func registerUser(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	_, err := db.Exec("INSERT INTO admin (username, password) VALUES (?, ?)", input.Username, string(hashedPassword))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User already exists or DB error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func loginUser(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	var hashedPassword string
	err := db.QueryRow("SELECT password FROM admin WHERE username = ?", input.Username).Scan(&hashedPassword)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

// ==== Student ====

func getStudents(c *gin.Context) {
	rows, err := db.Query("SELECT student_id, student_name, Age, Gender, class_id FROM student")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch students"})
		return
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.ID, &s.Name, &s.Age, &s.Gender, &s.Class); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse student data"})
			return
		}
		students = append(students, s)
	}
	c.JSON(http.StatusOK, students)
}

func addStudent(c *gin.Context) {
	var newStudent Student
	if err := c.BindJSON(&newStudent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	_, err := db.Exec("INSERT INTO student (student_name, Age, Gender, class_id) VALUES (?, ?, ?, ?)",
		newStudent.Name, newStudent.Age, newStudent.Gender, newStudent.Class)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student added successfully"})
}

func updateStudent(c *gin.Context) {
	id := c.Param("id")
	var s Student
	if err := c.BindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	_, err := db.Exec(`UPDATE student SET student_name=?, Age=?, Gender=?, class_id=? WHERE student_id=?`,
		s.Name, s.Age, s.Gender, s.Class, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student updated successfully"})
}

func deleteStudent(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM student WHERE student_id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted successfully"})
}

// ==== Marks ====

func getMarks(c *gin.Context) {
	rows, err := db.Query(`
		SELECT s.student_name, sub.subject_name, m.marks_obtained
		FROM marks m
		JOIN student s ON m.student_id = s.student_id
		JOIN subject sub ON m.subject_id = sub.subject_id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch marks"})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		var studentName, subjectName sql.NullString
		var marksObtained sql.NullInt64

		err := rows.Scan(&studentName, &subjectName, &marksObtained)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan marks", "details": err.Error()})
			return
		}

		// Build response map and handle NULLs gracefully
		result := map[string]interface{}{
			"student_name":   "",
			"subject_name":   "",
			"marks_obtained": nil,
		}
		if studentName.Valid {
			result["student_name"] = studentName.String
		}
		if subjectName.Valid {
			result["subject_name"] = subjectName.String
		}
		if marksObtained.Valid {
			result["marks_obtained"] = marksObtained.Int64
		}

		results = append(results, result)
	}

	c.JSON(http.StatusOK, results)
}

func addOrUpdateMark(c *gin.Context) {
	var mark MarkInput
	if err := c.BindJSON(&mark); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Check if the mark already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM marks WHERE student_id=? AND subject_id=?)", mark.StudentID, mark.SubjectID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check mark existence"})
		return
	}

	if exists {
		// Update mark
		_, err = db.Exec("UPDATE marks SET marks_obtained=? WHERE student_id=? AND subject_id=?",
			mark.MarksObtained, mark.StudentID, mark.SubjectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mark"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Mark updated successfully"})
	} else {
		// Add mark
		_, err = db.Exec("INSERT INTO marks (student_id, subject_id, marks_obtained) VALUES (?, ?, ?)",
			mark.StudentID, mark.SubjectID, mark.MarksObtained)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add mark"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Mark added successfully"})
	}
}

// ==== Subjects ====

func getSubjects(c *gin.Context) {
	rows, err := db.Query("SELECT subject_id, subject_name FROM subject")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subjects"})
		return
	}
	defer rows.Close()

	var subjects []Subject
	for rows.Next() {
		var s Subject
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan subjects"})
			return
		}
		subjects = append(subjects, s)
	}
	c.JSON(http.StatusOK, subjects)
}
