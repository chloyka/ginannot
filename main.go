package ginannot

import (
	"errors"
	"github.com/gin-gonic/gin"
	"reflect"
	"regexp"
	"strings"
)

type Handler interface{}

type GinAnnot struct {
	engine *gin.Engine
	logger Logger
}

type Options struct {
	Logger Logger
}

func New(r *gin.Engine, opts ...*Options) *GinAnnot {
	var logger Logger = &DefaultLogger{}
	if len(opts) > 0 && opts[0].Logger != nil {
		logger = opts[0].Logger
	}
	return &GinAnnot{
		engine: r,
		logger: logger,
	}
}

func (a *GinAnnot) Apply(controllers []Handler) {
	middlewaresMap := make(map[string]Middleware)
	middlewares := make(map[string][]func(*gin.Context))
	groups := make(map[string]*Group)

	for _, ctrl := range controllers {
		ctrlType := reflect.TypeOf(ctrl)
		ctrlVal := reflect.ValueOf(ctrl)

		if ctrlType.Kind() != reflect.Ptr {
			panic(errors.New("controller must be a pointer"))
		}

		ctrlTypeValue := ctrlType.Elem()
		ctrlValValue := ctrlVal.Elem()

		for i := 0; i < ctrlTypeValue.NumField(); i++ {
			fieldType := ctrlTypeValue.Field(i)
			fieldVal := ctrlValValue.Field(i)

			if fieldType.Type.Kind() != reflect.Struct {
				continue
			}

			for n := 0; n < fieldVal.NumField(); n++ {
				subfieldType := fieldType.Type.Field(n)

				if subfieldType.Type != reflect.TypeOf(Middleware{}) {
					continue
				}

				fieldTags := subfieldType.Tag.Get("middleware")
				if fieldTags == "" {
					continue
				}

				middleware, err := parseMiddlewareTag(fieldTags)
				if err != nil {
					return
				}

				handlerFunc := ctrlVal.MethodByName(subfieldType.Name)
				if !handlerFunc.IsValid() {
					continue
				}

				middleware.Callback = handlerFunc.Interface().(func(*gin.Context))
				middlewaresMap[middleware.Name] = *middleware
			}
		}
	}

	for key, middleware := range middlewaresMap {
		middlewares[key] = make([]func(*gin.Context), 0)

		for _, m := range middleware.Chain {
			if v, ok := middlewaresMap[m]; ok {
				middlewares[key] = append(middlewares[key], v.Callback)
			}
		}

		middlewares[key] = append(middlewares[key], middleware.Callback)
	}

	for _, ctrl := range controllers {
		ctrlType := reflect.TypeOf(ctrl)
		ctrlVal := reflect.ValueOf(ctrl)
		if ctrlType.Kind() != reflect.Ptr {
			panic(errors.New("controller must be a pointer"))
		}
		ctrlTypeValue := ctrlType.Elem()
		ctrlValValue := ctrlVal.Elem()

		for i := 0; i < ctrlTypeValue.NumField(); i++ {
			fieldType := ctrlTypeValue.Field(i)
			fieldVal := ctrlValValue.Field(i)
			if fieldType.Type == reflect.TypeOf(Group{}) {
				fieldTags := fieldType.Tag.Get("group")
				group := parseGroupTag(fieldTags)
				groupMiddlewares := parseMiddlewaresTag(fieldType.Tag.Get("middlewares"))
				group.Middlewares = groupMiddlewares
				groups[group.Name] = group
			}
			if fieldType.Type.Kind() == reflect.Struct {
				for n := 0; n < fieldVal.NumField(); n++ {
					subfieldType := fieldType.Type.Field(n)

					if subfieldType.Type == reflect.TypeOf(Group{}) {
						fieldTags := subfieldType.Tag.Get("group")
						group := parseGroupTag(fieldTags)

						groupMiddlewares := parseMiddlewaresTag(subfieldType.Tag.Get("middlewares"))
						group.Middlewares = groupMiddlewares
						groups[group.Name] = group
					}
				}
			}
		}
	}

	for _, ctrl := range controllers {
		ctrlType := reflect.TypeOf(ctrl)
		ctrlVal := reflect.ValueOf(ctrl)
		if ctrlType.Kind() != reflect.Ptr {
			panic(errors.New("controller must be a pointer"))
		}
		ctrlTypeValue := ctrlType.Elem()
		ctrlValValue := ctrlVal.Elem()

		for i := 0; i < ctrlTypeValue.NumField(); i++ {
			fieldType := ctrlTypeValue.Field(i)
			fieldVal := ctrlValValue.Field(i)

			firstChain := parseMiddlewaresTag(fieldType.Tag.Get("middlewares"))

			if fieldType.Type.Kind() == reflect.Struct {
				for n := 0; n < fieldVal.NumField(); n++ {
					subfieldType := fieldType.Type.Field(n)
					subfieldValue := fieldVal.Field(n)

					if subfieldType.Type == reflect.TypeOf(Route{}) {
						fieldTags := subfieldType.Tag.Get("gin")
						if fieldTags == "" {
							continue
						}
						tags := parseGinTag(fieldTags)
						subfieldValue.Set(reflect.ValueOf(*tags))
						httpMethod := strings.ToUpper(tags.Method)
						route := tags.Path

						handlerFunc := ctrlVal.MethodByName(subfieldType.Name)
						if !handlerFunc.IsValid() {
							continue
						}

						m := parseMiddlewaresTag(subfieldType.Tag.Get("middlewares"))
						flatten := make([]gin.HandlerFunc, 0)
						chain := make([]string, 0)
						for _, v := range firstChain {
							if _, ok := middlewares[v]; ok {
								fncs := middlewares[v]
								chain = append(chain, middlewaresMap[v].Name)
								for _, mvs := range middlewaresMap[v].Chain {
									if _, ok := middlewares[mvs]; ok {
										chain = append(chain, mvs)
									}
								}
								for _, f := range fncs {
									flatten = append(flatten, gin.HandlerFunc(f))
								}
							}
						}

						group := subfieldType.Tag.Get("group")
						if group != "" {
							groupRoute := group

							if _, ok := groups[group]; ok {
								groupRoute = groups[group].Path
								if groups[group].Middlewares != nil {
									for _, v := range groups[group].Middlewares {
										if _, ok := middlewares[v]; ok {
											fncs := middlewares[v]
											chain = append(chain, middlewaresMap[v].Name)
											for _, mvs := range middlewaresMap[v].Chain {
												if _, ok := middlewares[mvs]; ok {
													chain = append(chain, mvs)
												}
											}
											for _, f := range fncs {

												flatten = append(flatten, gin.HandlerFunc(f))
											}
										}
									}
								}
							}

							route = strings.TrimSuffix(groupRoute, "/") + "/" + strings.TrimPrefix(route, "/")
						}

						for _, v := range m {
							if _, ok := middlewares[v]; ok {
								fncs := middlewares[v]
								chain = append(chain, middlewaresMap[v].Name)
								for _, mvs := range middlewaresMap[v].Chain {
									if _, ok := middlewares[mvs]; ok {
										chain = append(chain, mvs)
									}
								}
								for _, f := range fncs {
									flatten = append(flatten, gin.HandlerFunc(f))
								}
							}
						}

						flatten = append(flatten, handlerFunc.Interface().(func(ctx *gin.Context)))
						a.logger.Info("registering route", httpMethod, route, " chain: "+strings.Join(chain, "->"))
						switch strings.ToLower(httpMethod) {
						case "get":
							a.engine.GET(route, flatten...)
						case "post":
							a.engine.POST(route, flatten...)
						case "put":
							a.engine.PUT(route, flatten...)
						case "patch":
							a.engine.PATCH(route, flatten...)
						case "delete":
							a.engine.DELETE(route, flatten...)
						case "options":
							a.engine.OPTIONS(route, flatten...)
						case "head":
							a.engine.HEAD(route, flatten...)
						case "any":
							a.engine.Any(route, flatten...)
						}
					}
				}
			}
		}
	}
}

type Route struct {
	Method string
	Path   string
}

type Group struct {
	Name        string
	Path        string
	Middlewares []string
}

func parseGinTag(tag string) *Route {
	var route Route
	r := regexp.MustCompile(`(\w+)\s+(.+)`)
	matches := r.FindStringSubmatch(tag)
	if len(matches) != 3 {
		route.Method = "GET"
		route.Path = tag

		return &route
	}

	route.Method = strings.ToUpper(matches[1])
	route.Path = matches[2]
	return &route
}

type Middleware struct {
	Name     string
	Callback gin.HandlerFunc
	Chain    []string
}

func parseMiddlewaresTag(tag string) []string {
	return strings.Split(tag, "->")
}

func parseGroupTag(tag string) *Group {
	group := Group{}
	for _, option := range strings.Split(tag, ",") {
		optionParts := strings.SplitN(option, "=", 2)
		switch optionParts[0] {
		case "name":
			group.Name = optionParts[1]
		case "path":
			group.Path = optionParts[1]
		}
	}

	return &group
}

func parseMiddlewareTag(tag string) (*Middleware, error) {
	info := Middleware{}
	for _, option := range strings.Split(tag, ",") {
		optionParts := strings.SplitN(option, "=", 2)
		switch optionParts[0] {
		case "name":
			info.Name = optionParts[1]
		case "chain":
			middlewares := strings.Split(optionParts[1], "->")
			for i := range middlewares {
				middlewares[i] = strings.TrimSpace(middlewares[i])
			}
			if len(middlewares) == 0 {
				middlewares = strings.Split(optionParts[1], "<-")
				for i := range middlewares {
					middlewares[i] = strings.TrimSpace(middlewares[i])
				}
				for i := 0; i < len(middlewares)/2; i++ {
					j := len(middlewares) - i - 1
					middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
				}
			}
			info.Chain = middlewares
		}
	}

	if info.Name == "" {
		return nil, errors.New("middleware name is required")
	}
	return &info, nil
}
