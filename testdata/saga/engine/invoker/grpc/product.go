package product

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

type server struct {
	UnimplementedProductInfoServer
}

func (*server) AddProduct(context.Context, *Product) (*ProductId, error) {
	log.Println("add product success")
	return &ProductId{Value: "1"}, nil
}
func (*server) GetProduct(context.Context, *ProductId) (*Product, error) {
	log.Println("get product success")
	return &Product{Id: "1"}, nil
}

func StartProductServer() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterProductInfoServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
