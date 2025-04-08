package main

import (
	"fmt"
	"reflect"
)

func main() {

	//	r := gin.Default()
	//	r.GET("/saf", func(c *gin.Context) {
	//		c.JSON(http.StatusOK, gin.H{})
	//		c.Next()
	//	})
	//
	//	http.HandleFunc("/sfa", indexHandler)
	//	http.ListenAndServe(":9999", nil)
	//t := ttst{}
	//for _, v := range t.sli {
	//	fmt.Println(v)
	//}
	//fmt.Println(len(t.sli))
	//
	//rand.Seed(time.Now().UnixNano())
	//randNum := rand.Intn(100)
	//fmt.Println(randNum)
	c := Calculate{}
	t := reflect.ValueOf(c)
	m := t.MethodByName("Add")
	if m.IsValid() {
		fmt.Println("有效")
		sli := []reflect.Value{reflect.ValueOf(10), reflect.ValueOf(20)}
		res := m.Call(sli)
		if len(res) > 0 {
			fmt.Println(res[0].Int())
		}
	}

}

type Calculate struct {
}

func (c *Calculate) Add(x, y int) int {
	return x + y
}

//type ttst struct {
//	sli []int
//}

//func indexHandler(w http.ResponseWriter, req *http.Request) {
//	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
//}
