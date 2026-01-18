# End-to-End Tutorial: Building a Multi-Entity OData Backend

This tutorial walks through building a complete OData backend with multiple entities that demonstrate relationships, lifecycle hooks, and custom operations. We will follow a practical "Products, Orders, and Customers" scenario using GORM for persistence and the `go-odata` service layer.

> **Tip:** The snippets below borrow heavily from the production-ready development server found in [`cmd/devserver`](../cmd/devserver/). Reusing those patterns gives you a proven foundation that already wires together migrations, seeding, lifecycle hooks, and custom actions/functions.

## Project Layout

Create a project structure that keeps the API entry point, domain models, and supporting packages cleanly separated:

```text
my-odata-backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── data/
│   │   ├── database.go
│   │   └── seed.go
│   └── odata/
│       ├── actions.go
│       ├── entities.go
│       └── functions.go
├── go.mod
└── go.sum
```

The rest of the tutorial fills in each file.

## Define the Entities and Relationships

Create `internal/odata/entities.go` and define entities that map to database tables. The sample below shows:

- Customers who place Orders
- Orders that contain many OrderItems
- OrderItems that point to Products
- Products that reference the Customer who created them and expose lifecycle hooks

```go
package odata

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "gorm.io/gorm"
)

type Customer struct {
    ID        uint      `json:"ID" gorm:"primaryKey" odata:"key"`
    Email     string    `json:"Email" gorm:"uniqueIndex;not null" odata:"required,maxlength=180"`
    Name      string    `json:"Name" gorm:"not null" odata:"required,maxlength=120,searchable"`
    CreatedAt time.Time `json:"CreatedAt"`

    Orders []Order `json:"Orders,omitempty" gorm:"foreignKey:CustomerID"`
}

type Product struct {
    ID          uint     `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string   `json:"Name" gorm:"not null" odata:"required,maxlength=100,searchable"`
    Description *string  `json:"Description" odata:"nullable,maxlength=500"`
    Price       float64  `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
    Status      int32    `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
    CreatedByID uint     `json:"CreatedByID" odata:"required"`

    CreatedBy *Customer    `json:"CreatedBy,omitempty" gorm:"foreignKey:CreatedByID"`
    Items     []OrderItem  `json:"Items,omitempty" gorm:"many2many:order_items"`
}

// ODataBeforeCreate enforces that only admins can create products.
// This mirrors the production hook in cmd/devserver/entities/product.go.
func (p Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    isAdmin := r.Header.Get("X-User-Role") == "admin"
    if !isAdmin {
        return fmt.Errorf("only administrators are allowed to create products")
    }
    return nil
}

// ODataBeforeUpdate reuses the same authorization guard as the development server.
func (p Product) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
    isAdmin := r.Header.Get("X-User-Role") == "admin"
    if !isAdmin {
        return fmt.Errorf("only administrators are allowed to update products")
    }
    return nil
}

type Order struct {
    ID         uint       `json:"ID" gorm:"primaryKey" odata:"key"`
    CustomerID uint       `json:"CustomerID" odata:"required"`
    PlacedAt   time.Time  `json:"PlacedAt" odata:"required"`
    Status     string     `json:"Status" gorm:"not null" odata:"required,enum=OrderStatus"`

    Customer  *Customer   `json:"Customer,omitempty"`
    LineItems []OrderItem `json:"LineItems,omitempty" gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
    OrderID   uint    `json:"OrderID" gorm:"primaryKey" odata:"key"`
    ProductID uint    `json:"ProductID" gorm:"primaryKey" odata:"key"`
    Quantity  int32   `json:"Quantity" gorm:"not null" odata:"required"`
    UnitPrice float64 `json:"UnitPrice" gorm:"not null" odata:"required,precision=10,scale=2"`

    Product *Product `json:"Product,omitempty"`
}

func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(&Customer{}, &Product{}, &Order{}, &OrderItem{})
}
```

The lifecycle hooks (`BeforeCreate`/`BeforeUpdate`) come directly from the development server’s `Product` entity and demonstrate how to enforce cross-cutting rules before GORM writes data.

## Database Initialization

Create `internal/data/database.go` to connect to either SQLite or PostgreSQL using the same pattern as [`cmd/devserver/main.go`](../cmd/devserver/main.go):

```go
package data

import (
    "fmt"

    "gorm.io/driver/postgres"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type Config struct {
    Driver string
    DSN    string
}

func OpenDatabase(cfg Config) (*gorm.DB, error) {
    switch cfg.Driver {
    case "postgres":
        if cfg.DSN == "" {
            return nil, fmt.Errorf("postgres DSN is required")
        }
        return gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
    case "sqlite", "":
        dsn := cfg.DSN
        if dsn == "" {
            dsn = "/tmp/go-odata-tutorial.db"
        }
        return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    default:
        return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
    }
}
```

This is nearly identical to the dev server’s flag-driven `switch` block and provides a simple configuration object for the rest of the tutorial.

## Seed the Database

Populate initial data so the API is useful immediately. The following `internal/data/seed.go` mirrors the production-ready `seedDatabase` helper in [`cmd/devserver/reseed.go`](../cmd/devserver/reseed.go):

```go
package data

import (
    "time"

    "github.com/your/module/internal/odata"
    "gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
    // Start from a clean slate in development.
    if err := db.Migrator().DropTable(&odata.OrderItem{}, &odata.Order{}, &odata.Product{}, &odata.Customer{}); err != nil {
        return err
    }
    if err := odata.AutoMigrate(db); err != nil {
        return err
    }

    customers := []odata.Customer{{
        ID:        1,
        Email:     "ada@example.com",
        Name:      "Ada Lovelace",
        CreatedAt: time.Now().AddDate(0, 0, -7),
    }, {
        ID:        2,
        Email:     "grace@example.com",
        Name:      "Grace Hopper",
        CreatedAt: time.Now().AddDate(0, 0, -3),
    }}
    if err := db.Create(&customers).Error; err != nil {
        return err
    }

    products := []odata.Product{{
        ID:          1,
        Name:        "Laptop",
        Price:       1599.99,
        Status:      1, // In stock
        CreatedByID: 1,
    }, {
        ID:          2,
        Name:        "Mechanical Keyboard",
        Price:       129.95,
        Status:      3, // In stock + on sale
        CreatedByID: 2,
    }}
    if err := db.Create(&products).Error; err != nil {
        return err
    }

    orders := []odata.Order{{
        ID:         1001,
        CustomerID: 1,
        PlacedAt:   time.Now().AddDate(0, 0, -2),
        Status:     "Processing",
    }}
    if err := db.Create(&orders).Error; err != nil {
        return err
    }

    items := []odata.OrderItem{{
        OrderID:   1001,
        ProductID: 1,
        Quantity:  1,
        UnitPrice: 1599.99,
    }, {
        OrderID:   1001,
        ProductID: 2,
        Quantity:  1,
        UnitPrice: 129.95,
    }}
    if err := db.Create(&items).Error; err != nil {
        return err
    }

    return nil
}
```

## Wire Up the OData Service

Create `cmd/server/main.go` and bootstrap the HTTP server by reusing the dev server’s approach to registration and middleware:

```go
package main

import (
    "flag"
    "fmt"
    "log"
    "net/http"

    "github.com/nlstn/go-odata"
    "github.com/your/module/internal/data"
    tutorialodata "github.com/your/module/internal/odata"
)

func main() {
    driver := flag.String("db", "sqlite", "Database driver: sqlite or postgres")
    dsn := flag.String("dsn", "", "Connection string for the database")
    flag.Parse()

    db, err := data.OpenDatabase(data.Config{Driver: *driver, DSN: *dsn})
    if err != nil {
        log.Fatal(err)
    }

    if err := tutorialodata.AutoMigrate(db); err != nil {
        log.Fatal(err)
    }
    if err := data.Seed(db); err != nil {
        log.Fatal(err)
    }

    service, err := odata.NewService(db)
    if err != nil {
        log.Fatal(err)
    }
    if err := service.SetNamespace("TutorialService"); err != nil {
        log.Fatal(err)
    }

    if err := service.RegisterEntity(&tutorialodata.Customer{}); err != nil {
        log.Fatal(err)
    }
    if err := service.RegisterEntity(&tutorialodata.Product{}); err != nil {
        log.Fatal(err)
    }
    if err := service.RegisterEntity(&tutorialodata.Order{}); err != nil {
        log.Fatal(err)
    }
    if err := service.RegisterEntity(&tutorialodata.OrderItem{}); err != nil {
        log.Fatal(err)
    }

    tutorialodata.RegisterFunctions(service, db)
    tutorialodata.RegisterActions(service, db)

    mux := http.NewServeMux()
    mux.Handle("/", service)

    fmt.Println("🚀 Tutorial API running at http://localhost:8080")
    fmt.Println("Products:   http://localhost:8080/Products")
    fmt.Println("Orders:     http://localhost:8080/Orders")
    fmt.Println("Customers:  http://localhost:8080/Customers")

    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatal(err)
    }
}
```

The server setup above is adapted from [`cmd/devserver/main.go`](../cmd/devserver/main.go#L19-L114) and keeps the wiring concise: configure the database, migrate, seed, register entities, and expose the service through `http.ServeMux`.

## Custom Functions and Actions

Implement custom logic in `internal/odata/functions.go` and `internal/odata/actions.go`. These follow the same pattern used in [`cmd/devserver/actions_functions.go`](../cmd/devserver/actions_functions.go).

### Unbound Function: Top Customers by Spend

```go
package odata

import (
    "net/http"
    "reflect"

    "github.com/nlstn/go-odata"
    "gorm.io/gorm"
)

func RegisterFunctions(service *odata.Service, db *gorm.DB) {
    // GET /GetTopCustomers?count=3
    _ = service.RegisterFunction(odata.FunctionDefinition{
        Name:       "GetTopCustomers",
        IsBound:    false,
        Parameters: []odata.ParameterDefinition{{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true}},
        ReturnType: reflect.TypeOf([]Customer{}),
        Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
            count := params["count"].(int64)

            var customers []Customer
            if err := db.Raw(`
                SELECT c.*, SUM(oi.quantity * oi.unit_price) AS total
                FROM customers c
                JOIN orders o ON o.customer_id = c.id
                JOIN order_items oi ON oi.order_id = o.id
                GROUP BY c.id
                ORDER BY total DESC
                LIMIT ?
            `, count).Scan(&customers).Error; err != nil {
                return nil, err
            }

            return customers, nil
        },
    })
}
```

### Bound Action: Apply Discount to an Order Item

```go
package odata

import (
    "encoding/json"
    "fmt"
    "net/http"
    "reflect"

    "github.com/nlstn/go-odata"
    "gorm.io/gorm"
)

func RegisterActions(service *odata.Service, db *gorm.DB) {
    _ = service.RegisterAction(odata.ActionDefinition{
        Name:      "ApplyItemDiscount",
        IsBound:   true,
        EntitySet: "OrderItems",
        Parameters: []odata.ParameterDefinition{{
            Name:     "percentage",
            Type:     reflect.TypeOf(float64(0)),
            Required: true,
        }},
        ReturnType: reflect.TypeOf(OrderItem{}),
        Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
            percentage := params["percentage"].(float64)

            // Extract key values from the request path: /OrderItems(OrderID=...,ProductID=...)
            var orderID, productID uint
            if _, err := fmt.Sscanf(r.URL.Path, "/OrderItems(OrderID=%d,ProductID=%d)/ApplyItemDiscount", &orderID, &productID); err != nil {
                return fmt.Errorf("invalid order item key")
            }

            var item OrderItem
            if err := db.First(&item, "order_id = ? AND product_id = ?", orderID, productID).Error; err != nil {
                return err
            }

            item.UnitPrice = item.UnitPrice * (1 - percentage/100)
            if err := db.Save(&item).Error; err != nil {
                return err
            }

            w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
            return json.NewEncoder(w).Encode(map[string]any{
                "@odata.context": "$metadata#OrderItems/$entity",
                "value":          item,
            })
        },
    })
}
```

Register both helpers inside `cmd/server/main.go` right after creating the service:

```go
    tutorialodata.RegisterFunctions(service, db)
    tutorialodata.RegisterActions(service, db)
```

## Running the API

1. **Initialize the module** (once):
   ```bash
   go mod init github.com/your/module
   go mod tidy
   ```

2. **Start the API** using SQLite (default):
   ```bash
   go run ./cmd/server
   ```

   Or point to PostgreSQL:
   ```bash
   go run ./cmd/server -db postgres -dsn "postgresql://user:pass@localhost:5432/odata"
   ```

3. **Explore the service**:
   - `GET http://localhost:8080/` – service document
   - `GET http://localhost:8080/$metadata` – EDM schema
   - `GET http://localhost:8080/Orders?$expand=Customer,LineItems($expand=Product)` – hydrated order graph
   - `POST http://localhost:8080/OrderItems(OrderID=1001,ProductID=1)/ApplyItemDiscount` with body `{"percentage":10}`

Because the lifecycle hooks require an admin header, create or update products with:

```bash
curl -X POST http://localhost:8080/Products \
     -H 'Content-Type: application/json' \
     -H 'X-User-Role: admin' \
     -d '{"Name": "Webcam", "Price": 89.99, "CreatedByID": 1, "Status": 1}'
```

You now have a working OData backend that mirrors the capabilities of the full development server while staying focused on a multi-entity Products/Orders/Customers scenario.
