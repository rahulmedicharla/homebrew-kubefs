curl -X POST http://localhost:4000/signup -H "Content-Type: application/json" -d '{"email": "temp@gmail.com", "password": "temp123456", "confirm_password": "temp123456", "security_question": "What is your favorite color", "security_answer": "blue"}'