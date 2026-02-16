import { Component } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';

@Component({
  selector: 'app-signup',
  imports: [ReactiveFormsModule],
  templateUrl: './signup.html',
  styleUrl: './signup.css',
})
export class Signup {
  signupForm: FormGroup;
  errorMessage: string = '';

  constructor(
    private fb: FormBuilder,
    private router: Router
  ) {
    this.signupForm = this.fb.group({
      username: ['', [Validators.required, Validators.minLength(3)]],
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', [Validators.required]]
    });
  }

  async onSubmit() {
    const {username,email,password,confirmPassword} = this.signupForm.value;
    if (this.signupForm.invalid) {
      this.errorMessage = 'Please fill out all fields correctly.';
      return;
    }
    if(password != confirmPassword){
      this.errorMessage = 'Passwords do not match, please fix';
      return;
    }
    console.log('Information passed basic checks, sending request')
    try{
      const response = await fetch('http://localhost:8080/api/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({username, email, password})
      });
      if (response.ok) {
        this.errorMessage = '';
        console.log('Successfully registered with username ', { username });
        this.router.navigate(['/login']);
      } else {
        this.errorMessage = 'Error with registering';
        console.log('Register did not work with username ', { username });
        return;
      }
    } catch (error){
        console.log('Error with request');
        return;
    }
  this.router.navigate(['/login']);
  }
}
