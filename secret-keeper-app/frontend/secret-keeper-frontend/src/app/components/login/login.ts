import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { CryptoService } from '../../services/crypto.service';
import { KeyService } from '../../services/key.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-login',
  imports: [ReactiveFormsModule],
  templateUrl: './login.html',
  styleUrl: './login.css',
})
export class Login implements OnInit {
  loginForm: FormGroup;
  errorMessage: string = '';
  successMessage: string = '';
  validCredentials: [string, string][] = [];
  constructor(
    private http: HttpClient,
    private fb: FormBuilder,
    private router: Router,
    private route: ActivatedRoute,
    private cryptoService: CryptoService,
    private keyService: KeyService,
    private authService: AuthService,
  ) {
    this.loginForm = this.fb.group({
      username: ['', [Validators.required, Validators.minLength(3)]],
      password: ['', [Validators.required, Validators.minLength(8)]]
    });
  }

  ngOnInit() {
    // Show a confirmation message if the user just verified their email.
    this.route.queryParams.subscribe(params => {
      if (params['verified'] === 'true') {
        this.successMessage = 'Email verified! You can now log in.';
      }
    });
  }

  onSubmit() {
    console.log("On submit triggered");
    if (this.loginForm.valid) {
      const {username, password} = this.loginForm.value;
        this.http.post('http://localhost:8080/api/login', {username, password}, { withCredentials: true }).subscribe({
          next: async () => {
            this.errorMessage = '';
            this.successMessage = '';
            console.log('Login worked with username', username);
            await this.initKeyPair(username, password);
            this.router.navigate(['/messaging']);
          },
          error: (err) => {
            if (err.status === 403) {
              this.errorMessage = 'Please verify your email address before logging in. Check your inbox.';
            } else {
              this.errorMessage = 'Invalid username or password';
            }
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

  /**
   * After a successful login, derive the wrapping key from the user's password,
   * then either load+unwrap the existing RSA private key from the server, or
   * generate a fresh key pair and save it.
   */
  private async initKeyPair(username: string, password: string): Promise<void> {
    try {
      const wrappingKey = await this.cryptoService.deriveWrappingKey(password, username);

      let existingKeys: { public_key: string; encrypted_private_key: string } | null = null;
      try {
        existingKeys = await this.keyService.getKeys();
      } catch {
        // No keys saved yet — will generate below.
      }

      if (existingKeys?.public_key && existingKeys?.encrypted_private_key) {
        // Unwrap the stored private key.
        const privateKey = await this.cryptoService.unwrapPrivateKey(existingKeys.encrypted_private_key, wrappingKey);
        const publicKey = await this.cryptoService.importPublicKey(existingKeys.public_key);
        this.authService.setKeyPair(publicKey, privateKey);
        console.log('[Login] RSA key pair loaded from server.');
      } else {
        // Generate a fresh key pair and save it.
        const keyPair = await this.cryptoService.generateRsaKeyPair();
        const publicKeyB64 = await this.cryptoService.exportPublicKey(keyPair.publicKey);
        const wrappedPrivate = await this.cryptoService.wrapPrivateKey(keyPair.privateKey, wrappingKey);
        await this.keyService.saveKeys(publicKeyB64, wrappedPrivate);
        this.authService.setKeyPair(keyPair.publicKey, keyPair.privateKey);
        console.log('[Login] Fresh RSA key pair generated and saved.');
      }
    } catch (e) {
      console.error('[Login] Failed to initialise key pair:', e);
    }
  }

  goToSignup() {
    this.router.navigate(['/signup']);
  }
  goToForgotPassword() {
    this.router.navigate(['/reset-password']);
  }
}
