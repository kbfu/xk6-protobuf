package protobuf

import (
	"context"
	"log"
	"sync"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/encoding/protojson"

	"go.k6.io/k6/js/modules"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func init() {
	modules.Register("k6/x/protobuf", new(Protobuf))
}

type Protobuf struct{}

type ProtoFile struct {
	messageDesc protoreflect.MessageDescriptor
	// Object pool to reuse dynamicpb.Message instances and reduce memory allocations
	messagePool sync.Pool
}

func (p *Protobuf) Load(protoFilePath, lookupType string) ProtoFile {
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{},
	}

	files, err := compiler.Compile(context.Background(), protoFilePath)
	if err != nil {
		log.Fatal(err)
	}
	if files == nil {
		log.Fatal("No files were passed as arguments")
	}
	if len(files) == 0 {
		log.Fatal("Zero files were parsed")
	}

	messageDesc := files[0].Messages().ByName(protoreflect.Name(lookupType))

	return ProtoFile{
		messageDesc: messageDesc,
		messagePool: sync.Pool{
			New: func() interface{} {
				return dynamicpb.NewMessage(messageDesc)
			},
		},
	}
}

func (p *ProtoFile) Encode(data string) []byte {
	// Get message from pool - Unmarshal will overwrite all fields
	dynamicMessage := p.messagePool.Get().(*dynamicpb.Message)
	defer p.messagePool.Put(dynamicMessage)

	err := protojson.Unmarshal([]byte(data), dynamicMessage)

	if err != nil {
		log.Fatal(err)
	}

	encodedBytes, err := proto.Marshal(dynamicMessage)
	if err != nil {
		log.Fatal(err)
	}

	return encodedBytes
}

func (p *ProtoFile) Decode(decodedBytes []byte) string {
	// Get message from pool - Unmarshal will overwrite all fields
	decodedMessage := p.messagePool.Get().(*dynamicpb.Message)
	defer p.messagePool.Put(decodedMessage)

	err := proto.Unmarshal(decodedBytes, decodedMessage)
	if err != nil {
		log.Fatal(err)
	}

	marshalOptions := protojson.MarshalOptions{
		UseProtoNames: true,
	}

	jsonString, err := marshalOptions.Marshal(decodedMessage)
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonString)
}
