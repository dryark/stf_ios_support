TARGET = ../bin/coordinator

all: $(TARGET)

$(TARGET): coordinator.go go.sum
	go build  -o $(TARGET) .

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum