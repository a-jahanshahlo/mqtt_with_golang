import { Component } from '@angular/core';
import { MonitoringServiceService } from './monitoring-service.service';
@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  title = 'carriot';
  tempLogs: any[] = [];
  warnings: any[] = [];

  messages: string[] = [];
  constructor(private monitor: MonitoringServiceService) { }
  self = this;
  ngOnInit(): void {
    console.log("evt.data")
    
    this.monitor.ws.onopen = function(evt) {
      console.log("Connection open...");
      
  };
  //Triggered when a message is received
  this.monitor.ws.onmessage = function(evt) {
      console.log("Received Message: " + evt.data);
  };
  //Triggered when the connection is closed
  this.monitor.ws.onclose = function(evt) {
      console.log("Connection closed.");
  };

  }
  sendMessage(message: string) {
    this.monitor.sendMessage(`{"deviceID":"1","deviceTime":"2016-07-16T19:20:30Z" , "latitude":"0", "longitude":"0", "altitude":"0", "course":"1", "satellites":"0", "speedOTG":"0", "accelerationX1": "0", "accelerationY1":"0" , "signal":"0", "powerSupply":"0"}`);
  }
}
