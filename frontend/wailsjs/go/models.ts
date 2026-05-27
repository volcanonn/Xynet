export namespace main {
	
	export class WgConfig {
	    name: string;
	    isAirvpn: boolean;
	    usageData?: string;
	
	    static createFrom(source: any = {}) {
	        return new WgConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isAirvpn = source["isAirvpn"];
	        this.usageData = source["usageData"];
	    }
	}

}

