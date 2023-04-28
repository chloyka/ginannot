package ginannot

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

type ProductController struct {
	GetProduct    Route `gin:"GET /product/:id" group:"v4"`
	CreateProduct Route `gin:"POST /product" group:"services"`
	UpdateProduct Route `gin:"PATCH /product/:id"`
	DeleteProduct Route `gin:"DELETE /product/:id"`
}

type Controller struct {
	Group `group:"name=v4,path=/api/v4"`
	ProductController
}

func (c *Controller) GetProduct(ctx *gin.Context) {
	ctx.String(200, "asd")
	//ctx.Status(418)
}

func (c *Controller) CreateProduct(ctx *gin.Context) {
	ctx.Status(422)
}

func (c *Controller) UpdateProduct(ctx *gin.Context) {
	// ...
}

func (c *Controller) DeleteProduct(ctx *gin.Context) {
	// ...
}

func TestApply(t *testing.T) {
	r := gin.New()
	c := &Controller{}
	annot := New(r)
	annot.Apply([]Handler{c})
	if len(r.Routes()) != 4 {
		t.Errorf("Expected 4 routes, got %d", len(r.Routes()))
	}

	routes := r.Routes()

	if routes[0].Path != "/api/v4/product/:id" || routes[0].Method != "GET" {
		t.Errorf("Expected /product/:id, got %s", routes[0].Path)
	}

	if routes[1].Path != "/services/product" || routes[1].Method != "POST" {
		t.Errorf("Expected /product, got %s", routes[1].Path)
	}

	if routes[2].Path != "/product/:id" || routes[2].Method != "PATCH" {
		t.Errorf("Expected /product/:id, got %s", routes[2].Path)
	}

	if routes[3].Path != "/product/:id" || routes[3].Method != "DELETE" {
		t.Errorf("Expected /product/:id, got %s", routes[3].Path)
	}
}

type Cont struct {
	Get Route `gin:"GET /product/:id" middlewares:"test5" group:"v4"`
}

type DefaultController struct {
	Group `group:"name=v4,path=/api/v7" middlewares:"test5->test3"`
	Cont  `middlewares:"test->test2"`
}

type Mid struct {
	Middleware  Middleware `middleware:"name=test"`
	Middleware2 Middleware `middleware:"name=test2,chain=test3->test4"`
	Middleware3 Middleware `middleware:"name=test3"`
	Middleware4 Middleware `middleware:"name=test4"`
	Middleware5 Middleware `middleware:"name=test5"`
}

type Middlewares struct {
	Mid
	someFunc func(ctx *gin.Context)
}

func (f *Middlewares) Middleware(ctx *gin.Context) {
	chain := []string{"test"}
	ctx.Set("chain", chain)
	ctx.Next()
	f.someFunc(ctx)
	res, _ := ctx.Get("chain")
	chain, _ = res.([]string)
	chain = append(chain, "test end")
	ctx.JSON(203, chain)
}

func (f *Middlewares) Middleware2(ctx *gin.Context) {
	res, _ := ctx.Get("chain")
	chain := res.([]string)
	chain = append(chain, "test2")
	ctx.Set("chain", chain)
	ctx.Next()
	res, _ = ctx.Get("chain")
	chain, _ = res.([]string)
	chain = append(chain, "test2 end")
	ctx.Set("chain", chain)
}

func (f *Middlewares) Middleware3(ctx *gin.Context) {
	res, _ := ctx.Get("chain")
	chain := res.([]string)
	chain = append(chain, "test3")
	ctx.Set("chain", chain)
	ctx.Next()
	res, _ = ctx.Get("chain")
	chain, _ = res.([]string)
	chain = append(chain, "test3 end")
	ctx.Set("chain", chain)
}

func (f *Middlewares) Middleware4(ctx *gin.Context) {
	res, _ := ctx.Get("chain")
	chain := res.([]string)
	chain = append(chain, "test4")
	ctx.Set("chain", chain)
	ctx.Next()
	res, _ = ctx.Get("chain")
	chain, _ = res.([]string)
	chain = append(chain, "test4 end")
	ctx.Set("chain", chain)
}

func (f *Middlewares) Middleware5(ctx *gin.Context) {
	res, _ := ctx.Get("chain")
	chain := res.([]string)
	chain = append(chain, "test5")
	ctx.Set("chain", chain)
	ctx.Next()
	res, _ = ctx.Get("chain")
	chain, _ = res.([]string)
	chain = append(chain, "test5 end")
	ctx.Set("chain", chain)
}

func (r *DefaultController) Get(ctx *gin.Context) {
	ctx.Status(203)
}

func TestWithMiddlewares(t *testing.T) {
	r := gin.New()
	m := &Middlewares{
		someFunc: func(ctx *gin.Context) {
			res, _ := ctx.Get("chain")
			chain := res.([]string)
			chain = append(chain, "some side effects")
			ctx.Set("chain", chain)
		},
	}
	c := &DefaultController{}
	annot := New(r)
	annot.Apply([]Handler{c, m})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v7/product/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != 203 {
		t.Errorf("Expected 203, got %d", w.Code)
	}
	if w.Body.String() != "[\"test\",\"test3\",\"test4\",\"test2\",\"test5\",\"test3\",\"test5\",\"test5 end\",\"test3 end\",\"test5 end\",\"test2 end\",\"test4 end\",\"test3 end\",\"some side effects\",\"test end\"]" {
		t.Errorf("Expected [\"test\",\"test3\",\"test4\",\"test2\",\"test5\",\"test3\",\"test5\",\"test5 end\",\"test3 end\",\"test5 end\",\"test2 end\",\"test4 end\",\"test3 end\",\"some side effects\",\"test end\"], got %s", w.Body.String())
	}
}
