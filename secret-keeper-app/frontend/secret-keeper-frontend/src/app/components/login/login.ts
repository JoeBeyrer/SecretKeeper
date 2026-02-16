import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';

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

  async onSubmit() {
    console.log("On submit triggered")  
    if (this.loginForm.valid) {
      const { username, password } = this.loginForm.value;
      try{
        const response = await fetch('http://localhost:8080/api/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        if (response.ok) {
          this.errorMessage = '';
          console.log('Login worked with username ', { username });
          this.router.navigate(['/messaging']);
        } else {
          this.errorMessage = 'Login Incorrect';
          console.log('Login did not work with username ', { username });
        }
      } catch (error){
          console.log('Error with request');
        }
    } else {
      this.errorMessage = 'Fill in all fields';
    }
  }

  goToSignup() {
    this.router.navigate(['/signup']);
  }
}
