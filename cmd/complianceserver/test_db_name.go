package main
import (
"fmt"
"gorm.io/gorm"
"gorm.io/driver/sqlite"
)
func testDBName() {
db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
fmt.Printf("db.Name(): %v\n", db.Name())
fmt.Printf("db.Dialector.Name(): %v\n", db.Dialector.Name())
}
