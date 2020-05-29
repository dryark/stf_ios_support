TARGET = ../bin/coordinator

all: $(TARGET)

coordinator_sources := $(wildcard *.go)

$(TARGET): $(coordinator_sources) go.sum
	go build -o $(TARGET) .

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum