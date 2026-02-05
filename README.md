# SecretKeeper
SecretKeeper is a privacy first messenger where all messages are end-to-end encrypted and secure. The intention is to provide users absolute privacy while sharing files, messages, and other media. No data is persistently stored on remote servers, adhering to a strict no-logs policy. Using this basic idea as the backbone of the app, we will explore additional security-based features that may be used to enhance the app such as file and data storage, disappearing messages, secure key exchange, and metadata minimization. The app may be expanded to include chat rooms, random chatting, secure forums, or other social media style features if time permits.

## Members
- Joe Beyrer  
- Christian Farese  
- Gabriel Kravets  
- David Edelstein

## Roles
For this project, the intention is for every team member to gain exposure to both front-end AngularJS development and backend GO programming. We do not plan to strictly adhere to the roles below.
As a temporary placeholder, the role assignments will be:
- Front-End Engineers: Joe Beyrer, Christian Farese
- Back-End Engineers: Gabriel Kravets, David Edelstein

## Project Setup
- Install Go using [here](https://go.dev/dl/)
- Run Go server using `go run main.go` from the secret-keeper-app/backend directory
- Run `go get github.com/rs/cors` to install CORS middleware

- Run `npm install -g @angular/cli` to install Angular CLI
- Install node.js [here](https://nodejs.org/en/download)
- Run `ng serve` from the secret-keeper-app/frontend/secret-keeper-frontend directory to run the website at http://localhost:4200
  - May need to run `npm install zone.js` if a Zone.js error is encountered

- To create a new component, `run ng g c components/[NAME]` from the secret-keeper-frontend directory