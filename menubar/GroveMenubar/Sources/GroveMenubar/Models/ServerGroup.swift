import Foundation

struct ServerGroup: Identifiable {
    let id: String
    let name: String
    let path: String
    var servers: [Server]

    var isRunning: Bool {
        servers.contains { $0.isRunning }
    }

    var runningCount: Int {
        servers.filter { $0.isRunning }.count
    }

    var totalCount: Int {
        servers.count
    }
}

class ServerGrouper {
    static func groupServers(_ servers: [Server]) -> [ServerGroup] {
        // Group servers by their main repo (project they belong to)
        var groups: [String: [Server]] = [:]

        for server in servers {
            let groupKey = extractGroupKey(from: server)
            groups[groupKey, default: []].append(server)
        }

        // Convert to ServerGroup objects
        return groups.map { key, servers in
            ServerGroup(
                id: key,
                name: extractGroupName(from: key),
                path: key,
                servers: sortServers(servers)
            )
        }.sorted { lhs, rhs in
            // Put groups with running servers first.
            if lhs.isRunning != rhs.isRunning {
                return lhs.isRunning
            }
            // Then prioritize groups with more active servers.
            if lhs.runningCount != rhs.runningCount {
                return lhs.runningCount > rhs.runningCount
            }
            return lhs.name < rhs.name
        }
    }

    private static func sortServers(_ servers: [Server]) -> [Server] {
        servers.sorted { lhs, rhs in
            // Running servers should be easy to find first.
            if lhs.isRunning != rhs.isRunning {
                return lhs.isRunning
            }
            // Then show active worktrees ahead of inactive ones.
            let lhsActive = lhs.hasClaude == true || lhs.hasVSCode == true || lhs.gitDirty == true
            let rhsActive = rhs.hasClaude == true || rhs.hasVSCode == true || rhs.gitDirty == true
            if lhsActive != rhsActive {
                return lhsActive
            }
            return lhs.name < rhs.name
        }
    }

    private static func extractGroupKey(from server: Server) -> String {
        // Prefer mainRepo if available (identifies the project)
        if let mainRepo = server.mainRepo, !mainRepo.isEmpty {
            return mainRepo
        }

        // Fallback to parent directory
        let url = URL(fileURLWithPath: server.path)
        let parentPath = url.deletingLastPathComponent().path

        // If the parent is the home directory or root, use the path itself
        let homeDir = NSHomeDirectory()
        if parentPath == homeDir || parentPath == "/" {
            return server.path
        }

        return parentPath
    }

    private static func extractGroupName(from path: String) -> String {
        // Extract a friendly name from the path (last component = project name)
        let url = URL(fileURLWithPath: path)
        return url.lastPathComponent
    }

    // Check if servers should be grouped (only group if there are multiple groups)
    static func shouldGroup(_ servers: [Server]) -> Bool {
        let groups = groupServers(servers)
        return groups.count > 1
    }
}

// UserDefaults extension for collapsed groups
class CollapsedGroupsManager {
    static let shared = CollapsedGroupsManager()
    private let defaults = UserDefaults.standard
    private let key = "collapsedServerGroups"

    private init() {}

    func isCollapsed(_ groupId: String) -> Bool {
        let collapsed = defaults.stringArray(forKey: key) ?? []
        return collapsed.contains(groupId)
    }

    func setCollapsed(_ groupId: String, collapsed: Bool) {
        var collapsedGroups = defaults.stringArray(forKey: key) ?? []

        if collapsed {
            if !collapsedGroups.contains(groupId) {
                collapsedGroups.append(groupId)
            }
        } else {
            collapsedGroups.removeAll { $0 == groupId }
        }

        defaults.set(collapsedGroups, forKey: key)
    }

    func toggleCollapsed(_ groupId: String) {
        setCollapsed(groupId, collapsed: !isCollapsed(groupId))
    }
}
