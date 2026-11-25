package domain

// ClassroomUsage 是教室/教务系统进入网关后的业务对象
type ClassroomUsage struct {
	ClassroomID string `json:"classroom_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`                 //occupied/free/unknown
	CourseName  string `json:"course_name,omitempty"`  //课程名称
	TeacherName string `json:"teacher_name,omitempty"` //教师名称
	Timestamp   string `json:"timestamp"`
}

type EducationSnapshot struct {
	Classrooms []ClassroomUsage `json:"classrooms"`
}
