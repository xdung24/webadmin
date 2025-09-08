import { Component, EventEmitter, Input, Output, ViewEncapsulation, ViewChild, OnInit } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatToolbarModule } from '@angular/material/toolbar';
import screenfull from 'screenfull';

import { Branding } from '../widgets/branding';
import { NotificationButton, NotificationMessage } from '../widgets/notification-button';
import { TranslateButton } from '../widgets/translate-button';
import { UserButton } from '../widgets/user-button';

@Component({
  selector: 'app-header',
  templateUrl: './header.html',
  styleUrl: './header.scss',
  host: {
    class: 'matero-header',
  },
  encapsulation: ViewEncapsulation.None,
  imports: [
    MatToolbarModule,
    MatButtonModule,
    MatIconModule,
    Branding,
    NotificationButton,
    TranslateButton,
    UserButton,
  ],
})
export class Header implements OnInit {
  @Input() showToggle = true;
  @Input() showBranding = false;

  @Output() toggleSidenav = new EventEmitter<void>();
  @Output() toggleSidenavNotice = new EventEmitter<void>();

  @ViewChild(NotificationButton) notificationButton!: NotificationButton;

  notifications: NotificationMessage[] = [];

  ngOnInit() {
    // Initialize with some sample notifications
    this.notifications = [
      {
        id: '1',
        title: 'Welcome to the Admin Panel',
        message: 'Your account has been successfully activated.',
        timestamp: new Date(Date.now() - 5 * 60 * 1000), // 5 minutes ago
        read: false,
        type: 'success'
      },
      {
        id: '2',
        title: 'System Maintenance',
        message: 'Scheduled maintenance will occur tonight at 2 AM.',
        timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
        read: false,
        type: 'warning'
      }
    ];
  }

  // Method to add new notifications (can be called from parent components)
  addNotification(notification: NotificationMessage) {
    this.notifications.unshift({
      id: Date.now().toString(),
      timestamp: new Date(),
      read: false,
      type: 'info',
      ...notification
    });
  }

  // Example method to simulate receiving a new notification
  simulateNewNotification() {
    const notifications = [
      { title: 'New Message', message: 'You have received a new message from admin.', type: 'info' as const },
      { title: 'Task Complete', message: 'Your data export has finished successfully.', type: 'success' as const },
      { title: 'Warning', message: 'Your session will expire in 5 minutes.', type: 'warning' as const },
      { title: 'Error', message: 'Failed to save your changes. Please try again.', type: 'error' as const }
    ];
    
    const randomNotification = notifications[Math.floor(Math.random() * notifications.length)];
    this.addNotification(randomNotification);
  }

  toggleFullscreen() {
    if (screenfull.isEnabled) {
      screenfull.toggle();
    }
  }
}
