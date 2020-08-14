TARGET = ../bin/coordinator

all: $(TARGET)

coordinator_sources := $(wildcard *.go)

$(TARGET): $(coordinator_sources) go.sum
	go build -o $(TARGET) -ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.GitDate=$(GIT_DATE) -X main.GitRemote=$(GIT_REMOTE) -X main.EasyVersion=$(EASY_VERSION)" . 

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum