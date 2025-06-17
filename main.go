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

	// Teacher APIs
	router.POST("/teachers", addTeacher)
	router.POST("/assign-teacher", assignTeacher)
	router.GET("/teacher-assignments", getTeacherAssignments)
	router.GET("/classrooms", GetAllClassrooms)
	router.GET("/teachers", GetAllTeachers)

	router.Run(":9091")
}

// ==========================
// Structs (with fixed tags)
// ==========================

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

type Teacher struct {
	TeacherID             int    `json:"teacher_id"`
	Name                  string `json:"name"`
	SubjectSpecialization string `json:"subject_specialization"`
}

type Teaches struct {
	TeacherID int `json:"teacher_id"`
	SubjectID int `json:"subject_id"`
	ClassID   int `json:"class_id"`
}

// ==========
// Auth APIs
// ==========

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

// ==================
// Student APIs
// ==================

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
	_, err := db.Exec("UPDATE student SET student_name=?, Age=?, Gender=?, class_id=? WHERE student_id=?",
		s.Name, s.Age, s.Gender, s.Class, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student updated successfully"})
}

func deleteStudent(c *gin.Context) {
	id := c.Param("id")
	fmt.Println("Trying to delete student ID:", id)

	res, err := db.Exec("DELETE FROM student WHERE student_id=?", id)
	if err != nil {
		fmt.Println("Error while deleting:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No student found with the given ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Student deleted successfully"})
}

// ==============
// Marks APIs
// ==============

func getMarks(c *gin.Context) {
	query := `
	SELECT s.student_name, sub.subject_name, m.marks_obtained
	FROM marks m
	JOIN student s ON m.student_id = s.student_id
	JOIN subject sub ON m.subject_id = sub.subject_id
	`

	rows, err := db.Query(query)
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

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM marks WHERE student_id=? AND subject_id=?)", mark.StudentID, mark.SubjectID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check mark existence"})
		return
	}

	if exists {
		_, err = db.Exec("UPDATE marks SET marks_obtained=? WHERE student_id=? AND subject_id=?",
			mark.MarksObtained, mark.StudentID, mark.SubjectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mark"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Mark updated successfully"})
	} else {
		_, err = db.Exec("INSERT INTO marks (student_id, subject_id, marks_obtained) VALUES (?, ?, ?)",
			mark.StudentID, mark.SubjectID, mark.MarksObtained)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add mark"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Mark added successfully"})
	}
}

// ================
// Subjects API
// ================

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

// ================
// Teacher APIs
// ================

func addTeacher(c *gin.Context) {
	var teacher Teacher
	if err := c.BindJSON(&teacher); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := db.Exec("INSERT INTO teacher (name, subject_specialization) VALUES (?, ?)",
		teacher.Name, teacher.SubjectSpecialization)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher added successfully"})
}

func assignTeacher(c *gin.Context) {
	var req struct {
		TeacherID int `json:"teacher_id"`
		SubjectID int `json:"subject_id"`
		ClassID   int `json:"class_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := db.Exec("INSERT INTO teaches (teacher_id, subject_id, class_id) VALUES (?, ?, ?)", req.TeacherID, req.SubjectID, req.ClassID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign teacher"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Teacher assigned successfully"})
}


func getTeacherAssignments(c *gin.Context) {
	query := `
	SELECT t.name AS teacher_name, s.subject_name, c.class_name
	FROM teaches te
	JOIN teacher t ON te.teacher_id = t.teacher_id
	JOIN subject s ON te.subject_id = s.subject_id
	JOIN classroom c ON te.class_id = c.class_id
	`
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teacher assignments"})
		return
	}
	defer rows.Close()

	type TeacherAssignment struct {
		TeacherName string `json:"teacher_name"`
		SubjectName string `json:"subject_name"`
		ClassName   string `json:"class_name"`
	}

	var result []TeacherAssignment
	for rows.Next() {
		var ta TeacherAssignment
		if err := rows.Scan(&ta.TeacherName, &ta.SubjectName, &ta.ClassName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse teacher assignments"})
			return
		}
		result = append(result, ta)
	}

	c.JSON(http.StatusOK, result)
}

// ===============
// Utility APIs
// ===============

func GetAllTeachers(c *gin.Context) {
	rows, err := db.Query("SELECT teacher_id, name FROM teacher")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch teachers", "details": err.Error()})
		return
	}
	defer rows.Close()

	var teachers []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		teachers = append(teachers, gin.H{
			"teacher_id": id,
			"name":       name,
		})
	}

	c.JSON(200, teachers)
}

func GetAllClassrooms(c *gin.Context) {
	rows, err := db.Query("SELECT class_id, class_name FROM classroom")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch classrooms", "details": err.Error()})
		return
	}
	defer rows.Close()

	var classes []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		classes = append(classes, gin.H{
			"class_id":   id,
			"class_name": name,
		})
	}

	c.JSON(200, classes)
}
