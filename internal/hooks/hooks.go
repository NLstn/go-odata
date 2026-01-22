package hooks

// Package hooks provides documentation for optional entity lifecycle hooks.
// Entity types can implement hook methods that are automatically discovered via reflection.
//
// All hook methods are optional. The library detects their presence using reflection and
// invokes them at the appropriate time in the request lifecycle. There is no interface
// that entities must implement - simply define any subset of the hook methods you need.
//
// For complete hook documentation, see the odata.EntityHook interface in the parent package.
//
// # Lifecycle Hook Methods
//
// Hook methods can be defined with either value or pointer receivers, depending on whether
// they need to modify the entity. Lifecycle hooks typically use pointer receivers since they
// often modify entity fields:
//
//	type Product struct {
//	    ID    uint    `json:"ID" odata:"key"`
//	    Name  string  `json:"Name"`
//	    Price float64 `json:"Price"`
//	}
//
//	// ODataBeforeCreate is called before creating a new Product
//	func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
//	    // Check if user is admin
//	    if !isAdmin(r) {
//	        return fmt.Errorf("only admins can create products")
//	    }
//	    return nil
//	}
//
// Available lifecycle hooks:
//   - ODataBeforeCreate(ctx context.Context, r *http.Request) error
//   - ODataAfterCreate(ctx context.Context, r *http.Request) error
//   - ODataBeforeUpdate(ctx context.Context, r *http.Request) error
//   - ODataAfterUpdate(ctx context.Context, r *http.Request) error
//   - ODataBeforeDelete(ctx context.Context, r *http.Request) error
//   - ODataAfterDelete(ctx context.Context, r *http.Request) error
//
// Additional optional read hooks can be implemented on entity types with the following signatures:
//
//  // ODataBeforeReadCollection lets you add GORM scopes to the underlying query before it is executed.
//  func (Product) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *odata.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
//
//  // ODataAfterReadCollection lets you replace or mutate the collection returned to the client.
//  func (Product) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *odata.QueryOptions, results interface{}) (interface{}, error)
//
//  // ODataBeforeReadEntity lets you add GORM scopes before reading a single entity.
//  func (Product) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *odata.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
//
//  // ODataAfterReadEntity lets you replace or mutate the entity returned to the client.
//  func (Product) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *odata.QueryOptions, entity interface{}) (interface{}, error)
//
// All read hooks receive the same context, HTTP request, and parsed OData query options that the handler uses.
// Before* hooks return additional GORM scopes to apply (`nil` means no extra scopes), while After* hooks
// receive the fetched data and can return a replacement value. In every case, returning a non-nil error aborts
// the request processing with that error.
