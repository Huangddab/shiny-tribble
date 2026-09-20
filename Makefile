APP_NAME=app.exe

run:
	air -- httpd

test:
	go run main.go bento -c conf/bento/basic.yaml

build:
	go build -o build/$(APP_NAME) .

clean:
	rm -f build/$(APP_NAME)
