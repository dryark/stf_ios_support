TARGET = ../bin/wda_wrapper

all: $(TARGET)

$(TARGET): wda_wrapper.go go.sum
	go build  -o $(TARGET) .

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum