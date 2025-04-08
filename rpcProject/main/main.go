package main

type Empty struct {
}
type Foo int

type Item struct {
	Name string
}

type Replys struct {
	ItemList []*Item
}

func (f Foo) Sum(args Empty, reply *Replys) error {

	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "666"})
	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "777"})
	return nil
}

type Goo int

func (f Goo) Del(args Empty, reply *Replys) error {

	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "777"})
	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "888"})
	return nil
}

func main() {

}
