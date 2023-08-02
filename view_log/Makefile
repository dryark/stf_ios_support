TARGET = ../view_log

all: $(TARGET)

$(TARGET): view_log.go go.sum
	go build -o $(TARGET) .

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum