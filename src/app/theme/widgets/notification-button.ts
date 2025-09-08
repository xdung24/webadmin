import { Component, Input } from '@angular/core';
import { MatBadgeModule } from '@angular/material/badge';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatMenuModule } from '@angular/material/menu';

export interface NotificationMessage {
  id?: string;
  title: string;
  message?: string;
  timestamp?: Date;
  read?: boolean;
  type?: 'info' | 'warning' | 'error' | 'success';
}

@Component({
  selector: 'app-notification',
  templateUrl: './notification-button.html',
  styleUrl: './notification-button.scss',
  imports: [MatBadgeModule, MatButtonModule, MatDividerModule, MatIconModule, MatListModule, MatMenuModule],
})

export class NotificationButton {
  @Input() messages: NotificationMessage[] = [];
  
  // Method to add a new message (can be called from parent component)
  addMessage(message: NotificationMessage) {
    this.messages.unshift({
      id: Date.now().toString(),
      timestamp: new Date(),
      read: false,
      type: 'info',
      ...message
    });
  }
  
  // Method to clear all messages
  clearMessages() {
    this.messages = [];
  }
  
  // Method to mark a message as read
  markAsRead(messageId: string) {
    const message = this.messages.find(m => m.id === messageId);
    if (message) {
      message.read = true;
    }
  }
  
  // Get unread message count
  get unreadCount(): number {
    return this.messages.filter(m => !m.read).length;
  }

  // Get icon based on message type
  getIconForType(type?: 'info' | 'warning' | 'error' | 'success'): string {
    switch (type) {
      case 'warning': return 'warning';
      case 'error': return 'error';
      case 'success': return 'check_circle';
      default: return 'info';
    }
  }

  // Format timestamp for display
  formatTimestamp(timestamp: Date): string {
    const now = new Date();
    const diff = now.getTime() - timestamp.getTime();
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return timestamp.toLocaleDateString();
  }
}
