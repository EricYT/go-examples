package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type Foo struct {
	Key    string        `gorm:"primary_key:true;column:key_me;size:32;not null"`
	Field1 sql.NullInt64 `gorm:"primary_key:true;column:field_1;AUTO_INCREMENT:false;not null"`
	Field2 []byte        `gorm:"column:field_2;type:blob"`
	Field3 string        `gorm:"column:field_3;type:varchar(32);default:'I am default value'"`
	Field4 time.Time     `gorm:"column:field_4"`
	Field5 *int          `gorm:"column:field_5"`
}

func (Foo) TableName() string { return "foo" }

func main() {
	fmt.Println("vim-go")
	// connect to database
	db, err := gorm.Open("mysql", "test:deepnet@tcp(localhost:3306)/test?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Fatalf("conecte to mysql error: %s\n", err)
	}
	defer db.Close()

	// log debug
	db.LogMode(true)

	// drop table
	log.Println("drop table: ", (Foo{}).TableName())
	db.DropTable(&Foo{})

	// check table is exists
	if !db.HasTable(&Foo{}) {
		log.Println("create table: ", (Foo{}).TableName())
		// migrate schema
		db.AutoMigrate(&Foo{})
	}

	// record is exist
	id := sql.NullInt64{Int64: 1, Valid: true}
	foo := &Foo{Key: "abc", Field1: id, Field2: []byte("hello,world"), Field4: time.Now()}
	if db.NewRecord(foo) {
		// new record
		log.Printf("record: %v primary key is blank", foo)
	} else {
		log.Printf("record: %v primary key is not blank", foo)
	}

	// insert record
	if err := db.Create(foo).Error; err != nil {
		log.Printf("record: %v insert record error: %s", foo, err)
	}
	log.Printf("record: %v key is blank: %t", foo, db.NewRecord(foo))

	// insert again
	if err := db.Create(foo).Error; err != nil {
		log.Printf("record: %v insert again error: %s", foo, err)
	}

	// select record
	id.Int64 = 2
	var f *Foo = &Foo{Key: "abc", Field1: id}
	if err := db.First(f).Error; err != nil {
		log.Printf("record get error: %s", err)
	} else {
		log.Printf("record found: %#v", f)
	}

	// select record by some conditions
	var foos []*Foo
	if err := db.Find(&foos).Error; err != nil {
		log.Printf("find all records error: %s", err)
	} else {
		spew.Dump(foos)
	}

	var f1s []*Foo
	id.Int64 = 3
	id.Valid = false
	if err := db.Where(&Foo{Field1: id}).Find(&f1s).Error; err != nil {
		log.Printf("find all records error: %s by field_1=2", err)
	} else {
		spew.Dump(f1s)
	}

	// update
	id.Int64 = 3
	if err := db.Model(Foo{}).UpdateColumns(Foo{Key: "abc", Field1: id, Field3: "balabala"}).Error; err != nil {
		log.Printf("update foo columns field3 error: %s", err)
	}

	log.Println("---------------------------------------------")
	id.Int64 = 1
	id.Valid = true
	ff := &Foo{Key: "abc", Field1: id, Field3: "yyyyyyyyyyy"}
	db.Save(ff)
	log.Println("+++++++++++++++++++++++++++++++++++++++++++++")
	db.Save(ff)
}
