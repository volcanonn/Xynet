export namespace main {
	
	export class AppSettings {
	    defaultInterface: string;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultInterface = source["defaultInterface"];
	        this.theme = source["theme"];
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
	
	export class ServiceStatus {
	    singboxRunning: boolean;
	    voponoCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.singboxRunning = source["singboxRunning"];
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

