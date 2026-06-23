export namespace main {
	
	export class AppSettings {
	    defaultInterface: string;
	    theme: string;
	    backend: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultInterface = source["defaultInterface"];
	        this.theme = source["theme"];
	        this.backend = source["backend"];
	    }
	}
	export class ImportedProxy {
	    name: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportedProxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.content = source["content"];
	    }
	}
	export class AppState {
	    canvasElements: any[];
	    proxies: ImportedProxy[];
	    settings: AppSettings;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canvasElements = source["canvasElements"];
	        this.proxies = this.convertValues(source["proxies"], ImportedProxy);
	        this.settings = this.convertValues(source["settings"], AppSettings);
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
	export class BackendStatus {
	    backend: string;
	    running: boolean;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new BackendStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backend = source["backend"];
	        this.running = source["running"];
	        this.status = source["status"];
	    }
	}
	export class DesktopApp {
	    name: string;
	    exec: string;
	    icon: string;
	    processName: string;
	
	    static createFrom(source: any = {}) {
	        return new DesktopApp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.exec = source["exec"];
	        this.icon = source["icon"];
	        this.processName = source["processName"];
	    }
	}
	
	export class RoutingRule {
	    processName: string;
	    tunnelId: string;
	    tunnelLabel: string;
	    tunnelType: string;
	
	    static createFrom(source: any = {}) {
	        return new RoutingRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.processName = source["processName"];
	        this.tunnelId = source["tunnelId"];
	        this.tunnelLabel = source["tunnelLabel"];
	        this.tunnelType = source["tunnelType"];
	    }
	}
	export class ServiceStatus {
	    backendRunning: boolean;
	    backendStatus: string;
	    backendName: string;
	    voponoCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backendRunning = source["backendRunning"];
	        this.backendStatus = source["backendStatus"];
	        this.backendName = source["backendName"];
	        this.voponoCount = source["voponoCount"];
	    }
	}
	export class VoponoProcess {
	    id: string;
	    appName: string;
	    configName: string;
	    pid: number;
	
	    static createFrom(source: any = {}) {
	        return new VoponoProcess(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.appName = source["appName"];
	        this.configName = source["configName"];
	        this.pid = source["pid"];
	    }
	}

}

