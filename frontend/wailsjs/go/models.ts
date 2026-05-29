export namespace main {
	
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

}

