import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { NotificationMessage } from '../../theme/widgets/notification-button';

@Injectable({
  providedIn: 'root'
})
export class NotificationService {
  private notificationsSubject = new BehaviorSubject<NotificationMessage[]>([]);
  public notifications$ = this.notificationsSubject.asObservable();

  constructor() {
    // Initialize with some default notifications if needed
    this.initializeDefaultNotifications();
  }

  private initializeDefaultNotifications() {
    const defaultNotifications: NotificationMessage[] = [
      {
        id: 'welcome-1',
        title: 'Welcome to the Admin Panel',
        message: 'Your account has been successfully activated.',
        timestamp: new Date(Date.now() - 5 * 60 * 1000), // 5 minutes ago
        read: false,
        type: 'success'
      }
    ];
    this.notificationsSubject.next(defaultNotifications);
  }

  // Get current notifications
  get notifications(): NotificationMessage[] {
    return this.notificationsSubject.value;
  }

  // Add a new notification
  addNotification(notification: Omit<NotificationMessage, 'id' | 'timestamp'>): void {
    const newNotification: NotificationMessage = {
      id: Date.now().toString() + Math.random().toString(36).substring(2),
      timestamp: new Date(),
      read: false,
      type: 'info',
      ...notification
    };

    const currentNotifications = this.notifications;
    const updatedNotifications = [newNotification, ...currentNotifications];
    this.notificationsSubject.next(updatedNotifications);
  }

  // Mark a notification as read
  markAsRead(messageId: string): void {
    const notifications = this.notifications.map(notification =>
      notification.id === messageId 
        ? { ...notification, read: true }
        : notification
    );
    this.notificationsSubject.next(notifications);
  }

  // Mark all notifications as read
  markAllAsRead(): void {
    const notifications = this.notifications.map(notification => ({
      ...notification,
      read: true
    }));
    this.notificationsSubject.next(notifications);
  }

  // Remove a specific notification
  removeNotification(messageId: string): void {
    const notifications = this.notifications.filter(
      notification => notification.id !== messageId
    );
    this.notificationsSubject.next(notifications);
  }

  // Clear all notifications
  clearAllNotifications(): void {
    this.notificationsSubject.next([]);
  }

  // Get unread count
  get unreadCount(): number {
    return this.notifications.filter(notification => !notification.read).length;
  }

  // Get unread count as observable
  get unreadCount$(): Observable<number> {
    return new Observable(observer => {
      this.notifications$.subscribe(notifications => {
        const count = notifications.filter(n => !n.read).length;
        observer.next(count);
      });
    });
  }

  // Convenience methods for different notification types
  addSuccessNotification(title: string, message?: string): void {
    this.addNotification({ title, message, type: 'success' });
  }

  addWarningNotification(title: string, message?: string): void {
    this.addNotification({ title, message, type: 'warning' });
  }

  addErrorNotification(title: string, message?: string): void {
    this.addNotification({ title, message, type: 'error' });
  }

  addInfoNotification(title: string, message?: string): void {
    this.addNotification({ title, message, type: 'info' });
  }
}
