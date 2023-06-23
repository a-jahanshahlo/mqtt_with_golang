import { Component } from '@angular/core';
import { MonitoringServiceService } from './monitoring-service.service';
@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  title = 'carriot';
  messages: string[] = [];
  constructor(private monitor: MonitoringServiceService) { }
  ngOnInit(): void {
    this.monitor.connect()
      .subscribe(evt => {
        this.messages.push(evt.data);
      });
  }
  sendMessage(message: string) {
    this.monitor.sendMessage(message);
  }
}
