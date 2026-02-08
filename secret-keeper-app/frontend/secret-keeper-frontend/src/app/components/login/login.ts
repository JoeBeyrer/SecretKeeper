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
      username: ['', [Validators.required]],
      password: ['', [Validators.required]]
    });
  }

  async ngOnInit() {
    try {
      const response = await fetch('/login.txt');
      const text = await response.text();
      const cleanText = text.replace(/^\uFEFF/, '').trim();
      
      this.validCredentials = JSON.parse(cleanText);
    } catch (error) {
      console.error('something went wrong loading the login text file', error);
    }
  }

  onSubmit() {
    if (this.loginForm.valid) {
      const { username, password } = this.loginForm.value;
      
      const isValid = this.validCredentials.some(
        ([validUser, validPass]) => validUser === username && validPass === password
      );
      
      if (isValid) {
        this.errorMessage = '';
        console.log('login worked with username ', { username });
        this.router.navigate(['/messaging']);
      } else {
        this.errorMessage = 'Login Incorrect';
        console.log('login did not work with username ', { username });
      }
    } else {
      this.errorMessage = 'Fill in all fields';
    }
  }
}
