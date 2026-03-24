import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';

interface ProfileData {
  username: string;
  email: string;
  display_name: string;
  bio: string;
  profile_picture_url: string;
}

@Component({
  selector: 'app-profile',
  imports: [ReactiveFormsModule],
  templateUrl: './profile.html',
  styleUrl: './profile.css',
})
export class Profile implements OnInit {
  profile: ProfileData | null = null;
  profileForm: FormGroup;
  accountForm: FormGroup;
  errorMessage: string = '';
  successMessage: string = '';
  accountErrorMessage: string = '';
  accountSuccessMessage: string = '';
  isLoading: boolean = true;
  isSaving: boolean = false;
  isSavingAccount: boolean = false;
  isUploadingPicture: boolean = false;

  constructor(
    private http: HttpClient,
    private fb: FormBuilder,
    private router: Router,
    private route: ActivatedRoute
  ) {
    this.profileForm = this.fb.group({
      display_name: ['', [Validators.maxLength(50)]],
      bio: ['', [Validators.maxLength(200)]],
    });

    this.accountForm = this.fb.group({
      new_username: ['', [Validators.minLength(3)]],
      new_email: ['', [Validators.email]],
      new_password: ['', [Validators.minLength(8)]],
      confirm_new_password: [''],
    });
  }

  ngOnInit(): void {
    this.loadProfile();
    this.route.queryParams.subscribe(params => {
      if (params['email_updated'] === 'true') {
        this.accountSuccessMessage = 'Email address updated successfully.';
      }
    });
  }

  loadProfile(): void {
    this.http.get<ProfileData>('http://localhost:8080/api/profile', { withCredentials: true }).subscribe({
      next: (profile) => {
        this.profile = profile;
        this.profileForm.setValue({
          display_name: profile.display_name || '',
          bio: profile.bio || '',
        });
        this.isLoading = false;
      },
      error: (err) => {
        if (err.status === 401) {
          this.router.navigate(['/login']);
        } else {
          this.errorMessage = 'Failed to load profile.';
          this.isLoading = false;
        }
      },
    });
  }

  onSave(): void {
    if (this.profileForm.invalid) {
      this.errorMessage = 'Display name max 50 characters, bio max 200 characters.';
      return;
    }

    this.isSaving = true;
    this.errorMessage = '';
    this.successMessage = '';

    this.http.put<{ message: string }>(
      'http://localhost:8080/api/profile/update',
      this.profileForm.value,
      { withCredentials: true }
    ).subscribe({
      next: (res) => {
        this.successMessage = res.message;
        this.isSaving = false;
        if (this.profile) {
          this.profile.display_name = this.profileForm.value.display_name;
          this.profile.bio = this.profileForm.value.bio;
        }
      },
      error: () => {
        this.errorMessage = 'Failed to save profile. Please try again.';
        this.isSaving = false;
      },
    });
  }

  onSaveAccount(): void {
    const { new_username, new_email, new_password, confirm_new_password } = this.accountForm.value;

    if (!new_username && !new_email && !new_password) {
      this.accountErrorMessage = 'Please fill in at least one field to update.';
      return;
    }

    if (new_password && new_password !== confirm_new_password) {
      this.accountErrorMessage = 'New passwords do not match.';
      return;
    }

    this.isSavingAccount = true;
    this.accountErrorMessage = '';
    this.accountSuccessMessage = '';

    const payload: any = {};
    if (new_username) payload.new_username = new_username;
    if (new_email) payload.new_email = new_email;
    if (new_password) {
      payload.new_password = new_password;
    }

    this.http.put<{ message: string }>(
      'http://localhost:8080/api/account',
      payload,
      { withCredentials: true }
    ).subscribe({
      next: (res) => {
        this.accountSuccessMessage = res.message;
        this.isSavingAccount = false;
        this.accountForm.reset();
        if (this.profile) {
          if (new_username) this.profile.username = new_username;
          if (new_email) this.profile.email = new_email;
        }
      },
      error: (err) => {
        if (err.status === 409) {
          this.accountErrorMessage = 'Username or email is already taken.';
        } else if (err.status === 400 && err.error === 'that is already your current email') {
          this.accountErrorMessage = 'That is already your current email.';
        } else {
          this.accountErrorMessage = 'Failed to update account. Please try again.';
        }
        this.isSavingAccount = false;
      },
    });
  }

  onPictureSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    // Reset input value so the same file triggers change event again after a remove.
    input.value = '';
    const allowed = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
    if (!allowed.includes(file.type)) {
      this.errorMessage = 'Only JPEG, PNG, GIF, and WebP images are accepted.';
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      this.errorMessage = 'Image must be under 2 MB.';
      return;
    }

    this.isUploadingPicture = true;
    this.errorMessage = '';
    this.successMessage = '';

    const formData = new FormData();
    formData.append('picture', file);

    this.http.post<{ message: string; profile_picture_url: string }>(
      'http://localhost:8080/api/profile/picture',
      formData,
      { withCredentials: true }
    ).subscribe({
      next: (res) => {
        if (this.profile) {
          this.profile.profile_picture_url = res.profile_picture_url;
        }
        this.successMessage = res.message;
        this.isUploadingPicture = false;
      },
      error: () => {
        this.errorMessage = 'Failed to upload picture. Please try again.';
        this.isUploadingPicture = false;
      },
    });
  }

  onRemovePicture(): void {
    this.http.put<{ message: string }>(
      'http://localhost:8080/api/profile/update',
      { display_name: this.profileForm.value.display_name, bio: this.profileForm.value.bio, clear_picture: true },
      { withCredentials: true }
    ).subscribe({
      next: () => {
        if (this.profile) {
          this.profile.profile_picture_url = '';
        }
        this.successMessage = 'Profile picture removed.';
      },
      error: () => {
        this.errorMessage = 'Failed to remove picture. Please try again.';
      },
    });
  }

  onLogout(): void {
    this.http.post('http://localhost:8080/api/logout', {}, { withCredentials: true }).subscribe({
      next: () => this.router.navigate(['/login']),
      error: () => this.router.navigate(['/login']),
    });
  }

  goToMessaging(): void {
    this.router.navigate(['/messaging']);
  }

  goTo(page: string): void {
    this.router.navigate(['/' + page]);
  }
}
