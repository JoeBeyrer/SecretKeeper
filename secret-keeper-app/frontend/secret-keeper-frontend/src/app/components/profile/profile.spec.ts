import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { HttpTestingController } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { Profile } from './profile';

describe('Profile', () => {
  let component: Profile;
  let fixture: ComponentFixture<Profile>;
  let httpMock: HttpTestingController;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  const mockProfile = {
    username: 'alice',
    email: 'alice@example.com',
    display_name: 'Alice',
    bio: 'Hello!',
    profile_picture_url: '',
  };

  beforeEach(async () => {
    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [Profile, ReactiveFormsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
        { provide: ActivatedRoute, useValue: { queryParams: of({}) } },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(Profile);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);

    // Trigger ngOnInit — this queues the HTTP request
    fixture.detectChanges();

    // HttpTestingController.flush() is synchronous: the subscription callback
    // runs immediately and updates component state (profile, isLoading, form).
    const req = httpMock.expectOne('http://localhost:8080/api/profile');
    req.flush(mockProfile);

    // Run change detection from the component's own CDRef downward.
    // This avoids the NG0100 parent-dirty-check that triggers when calling
    // fixture.detectChanges() a second time after state changed during the flush.
    fixture.changeDetectorRef.detectChanges();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    httpMock.verify();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load profile data on init', () => {
    expect(component.profile).toEqual(mockProfile);
    expect(component.isLoading).toBe(false);
  });

  it('should populate profileForm with loaded profile data', () => {
    expect(component.profileForm.value).toEqual({
      display_name: 'Alice',
      bio: 'Hello!',
    });
  });

  it('should navigate to /login when loadProfile returns 401', () => {
    component.profile = null;
    component.loadProfile();
    const req = httpMock.expectOne('http://localhost:8080/api/profile');
    req.flush('Unauthorized', { status: 401, statusText: 'Unauthorized' });
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });

  it('should set errorMessage when loadProfile returns non-401 error', () => {
    component.loadProfile();
    const req = httpMock.expectOne('http://localhost:8080/api/profile');
    req.flush('Server error', { status: 500, statusText: 'Internal Server Error' });
    expect(component.errorMessage).toBe('Failed to load profile.');
  });

  it('onSave() should set errorMessage when profileForm is invalid', () => {
    component.profileForm.setValue({ display_name: 'A'.repeat(51), bio: '' });
    component.onSave();
    expect(component.errorMessage).toBeTruthy();
    httpMock.expectNone('http://localhost:8080/api/profile/update');
  });

  it('onSave() should PUT profile and set successMessage on success', () => {
    component.profileForm.setValue({ display_name: 'Alice Updated', bio: 'New bio' });
    component.onSave();
    const req = httpMock.expectOne('http://localhost:8080/api/profile/update');
    expect(req.request.method).toBe('PUT');
    req.flush({ message: 'Profile updated!' });
    expect(component.successMessage).toBe('Profile updated!');
    expect(component.isSaving).toBe(false);
    expect(component.profile!.display_name).toBe('Alice Updated');
  });

  it('onSave() should set errorMessage on save failure', () => {
    component.profileForm.setValue({ display_name: 'Alice', bio: '' });
    component.onSave();
    const req = httpMock.expectOne('http://localhost:8080/api/profile/update');
    req.flush('error', { status: 500, statusText: 'Server Error' });
    expect(component.errorMessage).toBe('Failed to save profile. Please try again.');
  });

  it('onSaveAccount() should set accountErrorMessage when all fields are empty', () => {
    component.accountForm.setValue({ new_username: '', new_email: '', new_password: '', confirm_new_password: '' });
    component.onSaveAccount();
    expect(component.accountErrorMessage).toBeTruthy();
    httpMock.expectNone('http://localhost:8080/api/account');
  });

  it('onSaveAccount() should set accountErrorMessage when passwords do not match', () => {
    component.accountForm.setValue({ new_username: '', new_email: '', new_password: 'pass1234', confirm_new_password: 'different1' });
    component.onSaveAccount();
    expect(component.accountErrorMessage).toContain('do not match');
    httpMock.expectNone('http://localhost:8080/api/account');
  });

  it('onSaveAccount() should PUT account and reset form on success', () => {
    component.accountForm.setValue({ new_username: 'alice2', new_email: '', new_password: '', confirm_new_password: '' });
    component.onSaveAccount();
    const req = httpMock.expectOne('http://localhost:8080/api/account');
    expect(req.request.method).toBe('PUT');
    req.flush({ message: 'Account updated!' });
    expect(component.accountSuccessMessage).toBe('Account updated!');
    expect(component.profile!.username).toBe('alice2');
  });

  it('onSaveAccount() should set accountErrorMessage with 409 conflict', () => {
    component.accountForm.setValue({ new_username: 'taken', new_email: '', new_password: '', confirm_new_password: '' });
    component.onSaveAccount();
    const req = httpMock.expectOne('http://localhost:8080/api/account');
    req.flush('conflict', { status: 409, statusText: 'Conflict' });
    expect(component.accountErrorMessage).toContain('already taken');
  });

  it('onLogout() should POST to /api/logout and navigate to /login', () => {
    component.onLogout();
    const req = httpMock.expectOne('http://localhost:8080/api/logout');
    expect(req.request.method).toBe('POST');
    req.flush({});
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });

  it('onLogout() should navigate to /login even on logout error', () => {
    component.onLogout();
    const req = httpMock.expectOne('http://localhost:8080/api/logout');
    req.flush('error', { status: 500, statusText: 'Server Error' });
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });

  it('goToMessaging() should navigate to /messaging', () => {
    component.goToMessaging();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/messaging']);
  });

  it('should set accountSuccessMessage when email_updated query param is true', async () => {
    TestBed.resetTestingModule();
    routerSpy = { navigate: vi.fn() };
    await TestBed.configureTestingModule({
      imports: [Profile, ReactiveFormsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
        { provide: ActivatedRoute, useValue: { queryParams: of({ email_updated: 'true' }) } },
      ],
    }).compileComponents();

    const f = TestBed.createComponent(Profile);
    const c = f.componentInstance;
    const hm = TestBed.inject(HttpTestingController);

    f.detectChanges();
    const req = hm.expectOne('http://localhost:8080/api/profile');
    req.flush(mockProfile);
    f.changeDetectorRef.detectChanges();

    expect(c.accountSuccessMessage).toBe('Email address updated successfully.');
    hm.verify();
  });

  it('onPictureSelected() should set errorMessage for disallowed file type', () => {
    const file = new File(['content'], 'test.bmp', { type: 'image/bmp' });
    // DataTransfer is not available in jsdom — build a minimal FileList-like object instead
    const fakeFileList = Object.assign([file], { item: (i: number) => (i === 0 ? file : null) });
    const input = document.createElement('input');
    input.type = 'file';
    Object.defineProperty(input, 'files', { value: fakeFileList });
    const event = { target: input } as unknown as Event;
    component.onPictureSelected(event);
    expect(component.errorMessage).toContain('Only JPEG');
    httpMock.expectNone('http://localhost:8080/api/profile/picture');
  });
});
