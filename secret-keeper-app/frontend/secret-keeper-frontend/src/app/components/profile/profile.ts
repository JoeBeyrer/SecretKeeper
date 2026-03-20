import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
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
  errorMessage: string = '';
  successMessage: string = '';
  isLoading: boolean = true;
  isSaving: boolean = false;
  isUploadingPicture: boolean = false;

  constructor(
    private http: HttpClient,
    private fb: FormBuilder,
    private router: Router
  ) {
    this.profileForm = this.fb.group({
      display_name: ['', [Validators.maxLength(50)]],
      bio: ['', [Validators.maxLength(200)]],
    });
  }

  ngOnInit(): void {
    this.loadProfile();
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

  goToLogin(): void {
    this.router.navigate(['/login']);
  }

  goToMessaging(): void {
    this.router.navigate(['/messaging']);
  }
}
