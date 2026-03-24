import { Component } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'app-friends',
  templateUrl: './friends.html',
  styleUrl: './friends.css',
})
export class Friends {
  constructor(private router: Router) {}

  goTo(page: string): void {
    this.router.navigate(['/' + page]);
  }
}
