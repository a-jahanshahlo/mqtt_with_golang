import { Injectable } from '@angular/core';
import { Observable, Subject,Observer} from 'rxjs';
 

@Injectable({
  providedIn: 'root'
})
export class MonitoringServiceService {
  //private ws: WebSocket ;
  //private subject: Subject<any> ;
  public ws:WebSocket = new WebSocket("ws://localhost:8087/monitoring");
 
  constructor() { 

  }
 

  sendMessage(message: string) {
    //console.log(message);
    this.ws.send(message);
  }
}
