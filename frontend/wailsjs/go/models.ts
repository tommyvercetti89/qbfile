export namespace main {
	
	export class Peer {
	    peer_id: string;
	    username: string;
	    ip: string;
	    tcp_port: number;
	    public_key: number[];
	    // Go type: time
	    last_seen: any;
	    online: boolean;
	    color: string;
	    status: string;
	    is_wan: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Peer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peer_id = source["peer_id"];
	        this.username = source["username"];
	        this.ip = source["ip"];
	        this.tcp_port = source["tcp_port"];
	        this.public_key = source["public_key"];
	        this.last_seen = this.convertValues(source["last_seen"], null);
	        this.online = source["online"];
	        this.color = source["color"];
	        this.status = source["status"];
	        this.is_wan = source["is_wan"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TransferState {
	    id: string;
	    filename: string;
	    filesize: number;
	    bytes_sent: number;
	    bytes_recv: number;
	    speed_mb: number;
	    percent: number;
	    status: string;
	    peer_name: string;
	    is_sender: boolean;
	    local_path: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.filesize = source["filesize"];
	        this.bytes_sent = source["bytes_sent"];
	        this.bytes_recv = source["bytes_recv"];
	        this.speed_mb = source["speed_mb"];
	        this.percent = source["percent"];
	        this.status = source["status"];
	        this.peer_name = source["peer_name"];
	        this.is_sender = source["is_sender"];
	        this.local_path = source["local_path"];
	    }
	}

}

