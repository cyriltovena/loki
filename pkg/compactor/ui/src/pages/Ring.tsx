import { useEffect, useState, useRef, useCallback } from "react";
import { RingResponse, RingInstance } from "../types";
import { useBasename } from "../contexts/BasenameContext";
import { formatDistanceToNowStrict, formatISO } from "date-fns";

const formatRelativeTime = (timestamp: string) => {
  const date = new Date(timestamp);
  return `${formatDistanceToNowStrict(date)} ago`;
};

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp);
  return formatISO(date, { format: "extended" });
};

const getStateColors = (state: string) => {
  switch (state) {
    case "ACTIVE":
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200";
    case "LEAVING":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200";
    case "PENDING":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200";
    case "JOINING":
      return "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200";
    case "LEFT":
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200";
    default:
      return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200";
  }
};

const Ring = () => {
  const [ring, setRing] = useState<RingResponse | null>(null);
  const [error, setError] = useState<string>("");
  const [isForgetLoading, setIsForgetLoading] = useState(false);
  const [, setTimeUpdate] = useState(0); // Used to force re-render
  const [selectedInstances, setSelectedInstances] = useState<Set<string>>(
    new Set()
  );
  const [animatingInstances, setAnimatingInstances] = useState<Set<string>>(
    new Set()
  );
  const previousInstances = useRef<RingInstance[]>([]);
  const basename = useBasename();

  // Update relative times every second
  useEffect(() => {
    const timer = setInterval(() => {
      setTimeUpdate((n) => n + 1);
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const fetchRing = useCallback(async () => {
    try {
      const response = await fetch(`${basename}api/ring?tokens=true`, {
        headers: {
          Accept: "application/json",
        },
      });
      if (!response.ok) {
        throw new Error("Failed to fetch ring data");
      }
      const data = (await response.json()) as RingResponse;

      // Find instances that are new or removed
      const currentIds = new Set((data.shards || []).map((i) => i.id));
      const previousIds = new Set(previousInstances.current.map((i) => i.id));

      // Only update if there are actual changes
      const hasChanges =
        currentIds.size !== previousIds.size ||
        [...currentIds].some((id) => !previousIds.has(id));

      if (hasChanges) {
        const changedInstances = new Set([
          ...[...currentIds].filter((id) => !previousIds.has(id)), // New instances
          ...[...previousIds].filter((id) => !currentIds.has(id)), // Removed instances
        ]);

        setAnimatingInstances(changedInstances);
        // Clear animation flags after animation duration
        setTimeout(() => {
          setAnimatingInstances(new Set());
        }, 500);

        // Only reset selection if instances were removed
        const removedSelected = [...selectedInstances].some(
          (id) => !currentIds.has(id)
        );
        if (removedSelected) {
          setSelectedInstances(new Set());
        }
      }

      previousInstances.current = data.shards || [];
      setRing(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    }
  }, [basename, selectedInstances]);

  useEffect(() => {
    fetchRing();
    const interval = setInterval(fetchRing, 3000);
    return () => clearInterval(interval);
  }, [basename, fetchRing]);

  const handleForget = async () => {
    if (selectedInstances.size === 0) return;

    try {
      setIsForgetLoading(true);
      const formData = new FormData();
      selectedInstances.forEach((id) => {
        formData.append("forget", id);
      });

      await fetch(`${basename}api/ring`, {
        method: "POST",
        body: formData,
      });

      await fetchRing();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsForgetLoading(false);
    }
  };

  const toggleInstance = (instanceId: string) => {
    setSelectedInstances((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(instanceId)) {
        newSet.delete(instanceId);
      } else {
        newSet.add(instanceId);
      }
      return newSet;
    });
  };

  if (error) {
    return <div className="text-red-500 p-4">Error: {error}</div>;
  }

  if (!ring) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    );
  }

  const instancesByState = (ring.shards || []).reduce((acc, instance) => {
    acc[instance.state] = (acc[instance.state] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  return (
    <div className="px-4 py-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-4">
          <div className="text-sm font-medium text-gray-500 dark:text-gray-400">
            Last Ring Update
          </div>
          <div className="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
            {ring.now ? formatRelativeTime(ring.now) : "N/A"}
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-4">
          <div className="text-sm font-medium text-gray-500 dark:text-gray-400">
            Active Instances
          </div>
          <div className="mt-1 text-lg font-semibold text-green-600 dark:text-green-400">
            {instancesByState["ACTIVE"] || 0}
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-4">
          <div className="text-sm font-medium text-gray-500 dark:text-gray-400">
            Inactive Instances
          </div>
          <div className="mt-1 text-lg font-semibold text-red-600 dark:text-red-400">
            {Object.entries(instancesByState)
              .filter(([state]) => state !== "ACTIVE")
              .reduce((sum, [_, count]) => sum + count, 0)}
          </div>
        </div>
      </div>
      <div className="mb-4 flex justify-between items-center">
        <div className="text-sm text-gray-500 dark:text-gray-400">
          {selectedInstances.size} instance
          {selectedInstances.size !== 1 ? "s" : ""} selected
        </div>
        {selectedInstances.size > 0 && (
          <button
            onClick={handleForget}
            disabled={isForgetLoading}
            className="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isForgetLoading && (
              <div className="w-4 h-4 border-2 border-gray-400 dark:border-gray-500 border-t-transparent rounded-full animate-spin mr-2"></div>
            )}
            Forget Selected
          </button>
        )}
      </div>
      <div className="flex flex-col">
        <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
          <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
            <div className="shadow overflow-hidden border-b border-gray-200 dark:border-gray-700 sm:rounded-lg">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-800">
                  <tr>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                    >
                      <span className="sr-only">Select</span>
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                    >
                      ID
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                    >
                      State
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                    >
                      Address
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                    >
                      Zone
                    </th>
                    <th
                      scope="col"
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                    >
                      Last Heartbeat
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                  {(ring.shards || []).map((instance) => (
                    <tr
                      key={instance.id}
                      onClick={() => toggleInstance(instance.id)}
                      className={`
                        cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800
                        transition-all duration-500 ease-in-out
                        ${
                          animatingInstances.has(instance.id)
                            ? "animate-fade-in"
                            : ""
                        }
                      `}
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <input
                          type="checkbox"
                          checked={selectedInstances.has(instance.id)}
                          onChange={() => toggleInstance(instance.id)}
                          className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded cursor-pointer"
                          onClick={(e) => e.stopPropagation()}
                        />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                        {instance.id}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStateColors(
                            instance.state
                          )}`}
                        >
                          {instance.state}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                        {instance.address}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                        {instance.zone}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                        <span title={formatTimestamp(instance.timestamp)}>
                          {formatTimestamp(instance.timestamp)}
                          <span className="ml-2 text-xs text-gray-400">
                            ({formatRelativeTime(instance.timestamp)})
                          </span>
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Ring;
