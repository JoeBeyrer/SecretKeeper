import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-login',
  imports: [ReactiveFormsModule],
  templateUrl: './login.html',
  styleUrl: './login.css',
})
export class Login implements OnInit {
  loginForm: FormGroup;
  errorMessage: string = '';
  validCredentials: [string, string][] = [];
  constructor(
    private http: HttpClient,
    private fb: FormBuilder,
    private router: Router
  ) {
    this.loginForm = this.fb.group({
      username: ['', [Validators.required, Validators.minLength(3)]],
      password: ['', [Validators.required, Validators.minLength(8)]]
    });
  }

  async ngOnInit() {
    // try {
    //   const response = await fetch('/login.txt');
    //   const text = await response.text();
    //   const cleanText = text.replace(/^\uFEFF/, '').trim();
    //   this.validCredentials = JSON.parse(cleanText);
    // } catch (error) {
    //   console.error('something went wrong loading the login text file', error);
    // }
  }

  onSubmit() {
    console.log("On submit triggered");
    if (this.loginForm.valid) {
      const {username, password} = this.loginForm.value;
        this.http.post('http://localhost:8080/api/login', {username, password}).subscribe({
          next: () => { //successful request
            this.errorMessage = '';
            console.log('Login worked with username', username);
            this.router.navigate(['/messaging']);
          },
          error: (err) => {
            this.errorMessage = 'Invalid username or password';
            console.log('Error with request ',  err);
            return;
          },
          complete: () => {
            console.log('Login request finished');
          }
        });
    } else {
      this.errorMessage = 'Please fill in all fields with valid entries';
      console.log('Invalid loginForm');
    }
  }
  goToSignup() {
    this.router.navigate(['/signup']);
  }
}
