all: clean plugin

plugin: 
	go build -o ./dist/user ./user
	chmod +x ./dist/user

clean:
	rm -rf ./dist