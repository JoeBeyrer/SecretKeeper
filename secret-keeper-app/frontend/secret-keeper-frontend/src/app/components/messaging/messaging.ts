import { Component, NgZone, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';

interface Message {
  username: string;
  time: string;
  content: string;
}

@Component({
  selector: 'app-messaging',
  imports: [FormsModule],
  templateUrl: './messaging.html',
  styleUrl: './messaging.css',
})
export class Messaging implements OnInit {
  messages: Message[] = [];
  newMessage: string = '';
  currentUser: string = 'User1';
  errorMessage: string = '';

  constructor(private ngZone: NgZone) {}

  async ngOnInit() { //this one loads messages from messages.txt
    try {
      const response = await fetch('/messaging.txt');
      const text = await response.text();

      const lines = text.split('\n').filter(line => line.trim() !== '');

      const parsed = lines.map(line => {
        const fixed = line.replace(
          /,\s*(\d{1,2}:\d{2}:\d{2}\s+\d{1,2}-\d{1,2}-\d{4})\s*,/,
          ', "$1",' //i love regex <3
        );
        const arr = JSON.parse(fixed);
        return {
          username: arr[0],
          time: arr[1],
          content: arr[2]
        };
      });

      this.ngZone.run(() => {
        this.messages = parsed;
      });
    } catch (error) {
      console.error('Error loading messages.txt', error);
      this.ngZone.run(() => {
        this.errorMessage = 'Error loading messages.txt';
      });
    }
  }

  sendMessage() { //this one adds new messages to messages.txt
    if (!this.newMessage.trim()) return;

    const now = new Date();
    const hours = String(now.getUTCHours()).padStart(2, '0');
    const minutes = String(now.getUTCMinutes()).padStart(2, '0');
    const seconds = String(now.getUTCSeconds()).padStart(2, '0');
    const month = now.getUTCMonth() + 1;
    const day = now.getUTCDate();
    const year = now.getUTCFullYear();
    const timeStr = `${hours}:${minutes}:${seconds} ${month}-${day}-${year}`;

    this.messages.push({
      username: this.currentUser,
      time: timeStr,
      content: this.newMessage.trim()
    });

    this.newMessage = '';
  }
}
