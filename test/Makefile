all: test.proto test2.proto
	protoc --descriptor_set_out=descriptor_set test.proto test2.proto
	protoc --go_out=proto/prototmpl test.proto
	protoc --go_out=proto/something test2.proto
