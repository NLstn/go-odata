# OData Vocabulary Annotations

This guide explains how to add OData vocabulary annotations to your entities and properties. Annotations provide rich metadata that clients can use to understand your service better.

## Overview

OData vocabulary annotations allow you to add semantic metadata to your service elements (entity types, properties, entity sets, etc.). Common use cases include:

- Marking properties as computed (read-only)
- Adding human-readable descriptions
- Specifying immutable properties
- Defining permission models
- Describing capabilities and restrictions

## Standard OData Vocabularies

The library supports the standard OData vocabularies:

| Vocabulary | Namespace | Description |
|------------|-----------|-------------|
| Core | `Org.OData.Core.V1` | Basic terms like Computed, Immutable, Description |
| Capabilities | `Org.OData.Capabilities.V1` | Service capabilities (insert, update, delete restrictions) |
| Validation | `Org.OData.Validation.V1` | Input validation patterns and constraints |

## Adding Annotations via Struct Tags

The simplest way to add annotations is using the `annotation:` prefix in the `odata` struct tag:

```go
type Product struct {
    ID        uint    `json:"ID" gorm:"primaryKey"`
    Name      string  `json:"Name" odata:"required,annotation:Core.Description=Product display name"`
    SKU       string  `json:"SKU" odata:"annotation:Core.Immutable"`
    CreatedAt string  `json:"CreatedAt" odata:"auto,annotation:Core.Computed"`
    Price     float64 `json:"Price"`
}
```

### Annotation Tag Format

- Simple boolean annotation: `annotation:Core.Computed` (sets value to `true`)
- Annotation with value: `annotation:Core.Description=My description`
- Annotation with qualifier (hash): `annotation:Core.Description#Short=Short description`
- Annotation with qualifier (explicit): `annotation:Core.Description=Short description;qualifier=Short`
- Full namespace: `annotation:Org.OData.Core.V1.Computed`
- Short alias: `annotation:Core.Computed` (automatically expanded)

### Supported Aliases

| Alias | Full Namespace |
|-------|----------------|
| `Core.` | `Org.OData.Core.V1.` |
| `Capabilities.` | `Org.OData.Capabilities.V1.` |
| `Validation.` | `Org.OData.Validation.V1.` |

## Automatic Annotation Detection

Some annotations are automatically added based on property flags:

| Property Flag | Auto-Added Annotation |
|---------------|----------------------|
| `odata:"auto"` | `Org.OData.Core.V1.Computed` |
| Database-generated key (autoIncrement) | `Org.OData.Core.V1.Computed` |
| ETag property (`odata:"etag"`) | `Org.OData.Core.V1.Computed` |

Example:
```go
type Order struct {
    ID        uint      `json:"ID" gorm:"primaryKey;autoIncrement"` // Auto: Computed annotation
    Status    string    `json:"Status"`
    CreatedAt time.Time `json:"CreatedAt" odata:"auto"`             // Auto: Computed annotation
    Version   int       `json:"Version" odata:"etag"`               // Auto: Computed annotation
}
```

## Adding Annotations via API

You can also add annotations programmatically after registering entities:

### Entity-Level Annotations

```go
service, _ := odata.NewService(db)
service.RegisterEntity(&Product{})

// Add description to the entity type
err := service.RegisterEntityAnnotation("Products",
    "Org.OData.Core.V1.Description",
    "Product catalog items available for sale")
if err != nil {
    log.Fatal(err)
}
```

### Property-Level Annotations

```go
// Add description to a property
err := service.RegisterPropertyAnnotation("Products", "Name",
    "Org.OData.Core.V1.Description",
    "The product's display name shown to customers")
if err != nil {
    log.Fatal(err)
}

// Mark a property as computed
err = service.RegisterPropertyAnnotation("Products", "LastModified",
    "Org.OData.Core.V1.Computed",
    true)
if err != nil {
    log.Fatal(err)
}

// Set permission level
err = service.RegisterPropertyAnnotation("Products", "InternalNotes",
    "Org.OData.Core.V1.Permissions",
    "None")  // Hidden from clients
```

## Common Annotation Terms

### Core Vocabulary

| Term | Value Type | Description |
|------|------------|-------------|
| `Core.Computed` | Boolean | Property value is computed by the server |
| `Core.Immutable` | Boolean | Property cannot be changed after creation |
| `Core.Description` | String | Human-readable description |
| `Core.LongDescription` | String | Detailed description |
| `Core.Permissions` | String | "Read", "Write", "ReadWrite", or "None" |
| `Core.OptimisticConcurrency` | Array | Properties used for ETag computation |

### Capabilities Vocabulary

| Term | Value Type | Description |
|------|------------|-------------|
| `Capabilities.InsertRestrictions` | Object | Restrictions on creating entities |
| `Capabilities.UpdateRestrictions` | Object | Restrictions on updating entities |
| `Capabilities.DeleteRestrictions` | Object | Restrictions on deleting entities |
| `Capabilities.ReadRestrictions` | Object | Restrictions on reading entities |

## Viewing Annotations in Metadata

Annotations appear in the `$metadata` document in both XML and JSON formats.

### XML Metadata

```http
GET /$metadata
Accept: application/xml
```

```xml
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.01">
  <edmx:Reference Uri="https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.xml">
    <edmx:Include Namespace="Org.OData.Core.V1" Alias="Core" />
  </edmx:Reference>
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="ODataService">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID" /></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false" />
        <Property Name="Name" Type="Edm.String" Nullable="false" />
        <Property Name="CreatedAt" Type="Edm.DateTimeOffset" Nullable="true" />
      </EntityType>
      
      <!-- Annotations section -->
      <Annotations Target="ODataService.Product/Name">
        <Annotation Term="Org.OData.Core.V1.Description" String="Product display name" />
        <Annotation Term="Org.OData.Core.V1.Description" Qualifier="Short" String="Short name" />
      </Annotations>
      <Annotations Target="ODataService.Product/CreatedAt">
        <Annotation Term="Org.OData.Core.V1.Computed" Bool="true" />
      </Annotations>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>
```

### JSON Metadata (CSDL JSON)

```http
GET /$metadata
Accept: application/json
```

```json
{
  "$Version": "4.01",
  "$EntityContainer": "ODataService.Container",
  "ODataService": {
    "Product": {
      "$Kind": "EntityType",
      "$Key": ["ID"],
      "ID": {
        "$Type": "Edm.Int32"
      },
      "Name": {
        "$Type": "Edm.String",
        "@Org.OData.Core.V1.Description": "Product display name",
        "@Org.OData.Core.V1.Description#Short": "Short name"
      },
      "CreatedAt": {
        "$Type": "Edm.DateTimeOffset",
        "$Nullable": true,
        "@Org.OData.Core.V1.Computed": true
      }
    }
  }
}
```

## Best Practices

1. **Use Descriptions**: Add `Core.Description` annotations to help API consumers understand your model.

2. **Mark Computed Properties**: Use `Core.Computed` for server-generated fields like timestamps and auto-increment IDs.

3. **Immutable Properties**: Use `Core.Immutable` for properties that shouldn't change after creation (like external IDs or creation timestamps).

4. **Use Short Aliases**: Prefer `annotation:Core.Description` over the full namespace for readability.

5. **Combine with odata Tags**: Annotations work alongside other odata tags:
   ```go
   Name string `json:"Name" odata:"required,maxlength=100,annotation:Core.Description=Display name"`
   ```

## Instance Annotations in Payloads

Instance annotations are vocabulary annotations that appear in entity payloads (JSON responses) rather than just in metadata documents. The library automatically includes instance annotations in responses based on the metadata level.

### Metadata Levels and Annotations

The OData protocol defines three metadata levels that control what annotations appear in responses:

| Metadata Level | Control Annotations | Vocabulary Annotations | Example |
|---------------|---------------------|------------------------|---------|
| `full` | ✅ Included | ✅ Included | `?$format=application/json;odata.metadata=full` |
| `minimal` | ✅ Included | ❌ Excluded | `?$format=application/json;odata.metadata=minimal` (default) |
| `none` | ❌ Excluded | ❌ Excluded | `?$format=application/json;odata.metadata=none` |

**Control annotations** are OData-defined annotations like `@odata.context`, `@odata.id`, `@odata.etag`, etc.

**Vocabulary annotations** are semantic annotations from vocabularies like `Core`, `Capabilities`, etc.

### Instance Annotation Format

Instance annotations appear in JSON payloads with specific patterns:

- **Entity-level annotations**: Appear as `@<term>` immediately after `@odata.type`
- **Property-level annotations**: Appear as `<property>@<term>` immediately before the property value
- **Qualified annotations**: Append `#Qualifier` to the term (e.g., `@<term>#<Qualifier>` or `<property>@<term>#<Qualifier>`)

### Example Response with Full Metadata

```http
GET /Products(1)?$format=application/json;odata.metadata=full
```

```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "@odata.id": "Products(1)",
  "@odata.type": "#ODataService.Product",
  "@Org.OData.Core.V1.Description": "Product catalog item",
  "ID": 1,
  "Name@Org.OData.Core.V1.Description": "Product display name",
  "Name": "Widget",
  "CreatedAt@Org.OData.Core.V1.Computed": true,
  "CreatedAt": "2025-01-01T00:00:00Z",
  "Price": 99.99
}
```

### Annotations in Collection Responses

Instance annotations also appear in collection responses when using full metadata:

```http
GET /Products?$format=application/json;odata.metadata=full
```

```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "value": [
    {
      "@odata.id": "Products(1)",
      "@odata.type": "#ODataService.Product",
      "@Org.OData.Core.V1.Description": "Product catalog item",
      "ID": 1,
      "Name@Org.OData.Core.V1.Description": "Product display name",
      "Name": "Widget",
      "CreatedAt@Org.OData.Core.V1.Computed": true,
      "CreatedAt": "2025-01-01T00:00:00Z",
      "Price": 99.99
    }
  ]
}
```

## Instance Annotations in Requests

Clients can send instance annotations in POST and PATCH requests. The library ignores these annotations (they are not stored), but they are allowed per the OData specification.

### Example POST Request with Instance Annotations

```http
POST /Products
Content-Type: application/json

{
  "Name": "New Product",
  "Price": 49.99,
  "@Org.OData.Core.V1.Description": "Client annotation (ignored)",
  "Name@CustomVocab.Priority": "High (also ignored)"
}
```

The entity will be created with `Name="New Product"` and `Price=49.99`. The instance annotations are validated (to ensure they start with `@`) but are not persisted to the database.

### Special Annotation: @odata.bind

The `@odata.bind` annotation is a special OData control annotation used to bind navigation properties. Unlike vocabulary annotations, it is processed by the service:

```json
{
  "Name": "Order Item",
  "Product@odata.bind": "Products(1)",
  "Order@odata.bind": "Orders(123)"
}
```

## Best Practices

1. **Use Descriptions**: Add `Core.Description` annotations to help API consumers understand your model.

2. **Mark Computed Properties**: Use `Core.Computed` for server-generated fields like timestamps and auto-increment IDs.

3. **Immutable Properties**: Use `Core.Immutable` for properties that shouldn't change after creation (like external IDs or creation timestamps).

4. **Use Short Aliases**: Prefer `annotation:Core.Description` over the full namespace for readability.

5. **Combine with odata Tags**: Annotations work alongside other odata tags:
   ```go
   Name string `json:"Name" odata:"required,maxlength=100,annotation:Core.Description=Display name"`
   ```

6. **Use Full Metadata Sparingly**: Request full metadata (`odata.metadata=full`) only when you need the annotations, as it increases response size.

## See Also

- [OData Vocabularies Specification](https://docs.oasis-open.org/odata/odata/v4.01/os/vocabularies/)
- [Core Vocabulary](https://github.com/oasis-tcs/odata-vocabularies/blob/main/vocabularies/Org.OData.Core.V1.md)
- [Capabilities Vocabulary](https://github.com/oasis-tcs/odata-vocabularies/blob/main/vocabularies/Org.OData.Capabilities.V1.md)
- [OData JSON Format Specification](https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html)
