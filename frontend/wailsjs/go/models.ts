export namespace main {
	
	export class AddOfflineRequest {
	    urls: string[];
	    saveDirId: string;
	    saveDirPath: pan.Breadcrumb[];
	
	    static createFrom(source: any = {}) {
	        return new AddOfflineRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.saveDirId = source["saveDirId"];
	        this.saveDirPath = this.convertValues(source["saveDirPath"], pan.Breadcrumb);
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
	export class DirectoryTargetView {
	    id: string;
	    path: pan.Breadcrumb[];
	
	    static createFrom(source: any = {}) {
	        return new DirectoryTargetView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = this.convertValues(source["path"], pan.Breadcrumb);
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
	export class SettingsView {
	    preferredPlayer: string;
	    players: player.Status[];
	    configPath: string;
	    showTitleBadges: boolean;
	    smallFileFilterMB: number;
	    fileListDensity: string;
	    offlineRecentTargets: DirectoryTargetView[];
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preferredPlayer = source["preferredPlayer"];
	        this.players = this.convertValues(source["players"], player.Status);
	        this.configPath = source["configPath"];
	        this.showTitleBadges = source["showTitleBadges"];
	        this.smallFileFilterMB = source["smallFileFilterMB"];
	        this.fileListDensity = source["fileListDensity"];
	        this.offlineRecentTargets = this.convertValues(source["offlineRecentTargets"], DirectoryTargetView);
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
	export class BootstrapResult {
	    loggedIn: boolean;
	    user?: pan.UserView;
	    settings: SettingsView;
	    currentId: string;
	    proxyBase: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loggedIn = source["loggedIn"];
	        this.user = this.convertValues(source["user"], pan.UserView);
	        this.settings = this.convertValues(source["settings"], SettingsView);
	        this.currentId = source["currentId"];
	        this.proxyBase = source["proxyBase"];
	        this.version = source["version"];
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
	export class DeleteOfflineRequest {
	    hashes: string[];
	    deleteFiles: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeleteOfflineRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hashes = source["hashes"];
	        this.deleteFiles = source["deleteFiles"];
	    }
	}
	export class PlayRequest {
	    pickCode: string;
	    name: string;
	    startMs: number;
	    fromStart: boolean;
	    subtitle?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlayRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pickCode = source["pickCode"];
	        this.name = source["name"];
	        this.startMs = source["startMs"];
	        this.fromStart = source["fromStart"];
	        this.subtitle = source["subtitle"];
	    }
	}
	export class PlayResult {
	    playerId: string;
	    playerName: string;
	    path: string;
	    startMs: number;
	    resumeUsed: boolean;
	    subtitle?: string;
	    managedResume: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlayResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerId = source["playerId"];
	        this.playerName = source["playerName"];
	        this.path = source["path"];
	        this.startMs = source["startMs"];
	        this.resumeUsed = source["resumeUsed"];
	        this.subtitle = source["subtitle"];
	        this.managedResume = source["managedResume"];
	    }
	}
	export class PlaybackStateView {
	    pickCode: string;
	    resumeMs: number;
	    resumeText?: string;
	    subtitlePath?: string;
	    subtitleName?: string;
	    lastPlayedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaybackStateView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pickCode = source["pickCode"];
	        this.resumeMs = source["resumeMs"];
	        this.resumeText = source["resumeText"];
	        this.subtitlePath = source["subtitlePath"];
	        this.subtitleName = source["subtitleName"];
	        this.lastPlayedAt = source["lastPlayedAt"];
	    }
	}

}

export namespace pan {
	
	export class Breadcrumb {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Breadcrumb(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class FileItem {
	    fileId: string;
	    parentId: string;
	    name: string;
	    size: number;
	    pickCode: string;
	    isDirectory: boolean;
	    isVideo: boolean;
	    updatedAt: string;
	    durationSec?: number;
	    resumeMs?: number;
	    subtitlePath?: string;
	    lastPlayedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileId = source["fileId"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.pickCode = source["pickCode"];
	        this.isDirectory = source["isDirectory"];
	        this.isVideo = source["isVideo"];
	        this.updatedAt = source["updatedAt"];
	        this.durationSec = source["durationSec"];
	        this.resumeMs = source["resumeMs"];
	        this.subtitlePath = source["subtitlePath"];
	        this.lastPlayedAt = source["lastPlayedAt"];
	    }
	}
	export class DirectoryView {
	    dirId: string;
	    parentId: string;
	    name: string;
	    path: Breadcrumb[];
	    count: number;
	    offset: number;
	    limit: number;
	    hasMore: boolean;
	    items: FileItem[];
	
	    static createFrom(source: any = {}) {
	        return new DirectoryView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dirId = source["dirId"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.path = this.convertValues(source["path"], Breadcrumb);
	        this.count = source["count"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	        this.hasMore = source["hasMore"];
	        this.items = this.convertValues(source["items"], FileItem);
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
	
	export class LoginSessionView {
	    sessionId: string;
	    qrCodeDataUrl: string;
	    qrCodeContent: string;
	    expiresIn: number;
	    createdUnixSec: number;
	
	    static createFrom(source: any = {}) {
	        return new LoginSessionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.qrCodeDataUrl = source["qrCodeDataUrl"];
	        this.qrCodeContent = source["qrCodeContent"];
	        this.expiresIn = source["expiresIn"];
	        this.createdUnixSec = source["createdUnixSec"];
	    }
	}
	export class UserView {
	    userId: number;
	    userName: string;
	    faceUrl: string;
	    isVip: boolean;
	    vipLabel: string;
	    vipForever: boolean;
	    vipExpireAt: string;
	    spaceTotal: number;
	    spaceUsed: number;
	    spaceRemain: number;
	
	    static createFrom(source: any = {}) {
	        return new UserView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userId = source["userId"];
	        this.userName = source["userName"];
	        this.faceUrl = source["faceUrl"];
	        this.isVip = source["isVip"];
	        this.vipLabel = source["vipLabel"];
	        this.vipForever = source["vipForever"];
	        this.vipExpireAt = source["vipExpireAt"];
	        this.spaceTotal = source["spaceTotal"];
	        this.spaceUsed = source["spaceUsed"];
	        this.spaceRemain = source["spaceRemain"];
	    }
	}
	export class LoginStatusView {
	    state: string;
	    message: string;
	    loggedIn: boolean;
	    user?: UserView;
	
	    static createFrom(source: any = {}) {
	        return new LoginStatusView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.message = source["message"];
	        this.loggedIn = source["loggedIn"];
	        this.user = this.convertValues(source["user"], UserView);
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
	export class SearchResultView {
	    query: string;
	    count: number;
	    offset: number;
	    limit: number;
	    hasMore: boolean;
	    items: FileItem[];
	
	    static createFrom(source: any = {}) {
	        return new SearchResultView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.count = source["count"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	        this.hasMore = source["hasMore"];
	        this.items = this.convertValues(source["items"], FileItem);
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
	export class OfflineTaskView {
	    infoHash: string;
	    name: string;
	    size: number;
	    url: string;
	    addTime: string;
	    updateTime: string;
	    status: string;
	    statusCode: number;
	    statusGroup: string;
	    percent: number;
	    percentText: string;
	    speedText: string;
	    leftTimeText: string;
	    peers: number;
	    fileId: string;
	    deleteFileId: string;
	    dirId: string;
	
	    static createFrom(source: any = {}) {
	        return new OfflineTaskView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.infoHash = source["infoHash"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.url = source["url"];
	        this.addTime = source["addTime"];
	        this.updateTime = source["updateTime"];
	        this.status = source["status"];
	        this.statusCode = source["statusCode"];
	        this.statusGroup = source["statusGroup"];
	        this.percent = source["percent"];
	        this.percentText = source["percentText"];
	        this.speedText = source["speedText"];
	        this.leftTimeText = source["leftTimeText"];
	        this.peers = source["peers"];
	        this.fileId = source["fileId"];
	        this.deleteFileId = source["deleteFileId"];
	        this.dirId = source["dirId"];
	    }
	}
	export class OfflineListView {
	    quota: number;
	    total: number;
	    activeCount: number;
	    failedCount: number;
	    completedCount: number;
	    tasks: OfflineTaskView[];
	
	    static createFrom(source: any = {}) {
	        return new OfflineListView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.quota = source["quota"];
	        this.total = source["total"];
	        this.activeCount = source["activeCount"];
	        this.failedCount = source["failedCount"];
	        this.completedCount = source["completedCount"];
	        this.tasks = this.convertValues(source["tasks"], OfflineTaskView);
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
	

}

export namespace player {
	
	export class Status {
	    id: string;
	    name: string;
	    supported: boolean;
	    available: boolean;
	    disabled: boolean;
	    selected: boolean;
	    path?: string;
	    source?: string;
	    supportsStartPosition: boolean;
	    supportsSubtitle: boolean;
	    supportsManagedResume: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.supported = source["supported"];
	        this.available = source["available"];
	        this.disabled = source["disabled"];
	        this.selected = source["selected"];
	        this.path = source["path"];
	        this.source = source["source"];
	        this.supportsStartPosition = source["supportsStartPosition"];
	        this.supportsSubtitle = source["supportsSubtitle"];
	        this.supportsManagedResume = source["supportsManagedResume"];
	    }
	}

}
