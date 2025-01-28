import { useEffect, useState } from "react";
import {
  format,
  formatDistanceToNow,
  formatDuration,
  intervalToDuration,
} from "date-fns";
import { Link } from "react-router-dom";
import { DateWithHover } from "../components/DateWithHover";

interface DeleteRequest {
  request_id: string;
  start_time: number;
  end_time: number;
  query: string;
  status: string;
  created_at: number;
  user_id: string;
  deleted_lines: number;
}

const Deletes = () => {
  const [requests, setRequests] = useState<DeleteRequest[]>([]);
  const [selectedStatus, setSelectedStatus] = useState<string[]>(["received"]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = async (status: string) => {
    try {
      const response = await fetch(
        `/compactor/ui/api/deletes?status=${status}`
      );
      if (!response.ok) throw new Error("Failed to fetch delete requests");
      return await response.json();
    } catch (error) {
      console.error("Error fetching delete requests:", error);
      setError("Failed to fetch delete requests. Please try again.");
      return [];
    }
  };

  useEffect(() => {
    const fetchAllSelectedRequests = async () => {
      setLoading(true);
      setError(null);
      try {
        const allRequests = await Promise.all(
          selectedStatus.map((status) => fetchRequests(status))
        );
        const combined = allRequests
          .flat()
          .sort((a, b) => b.created_at - a.created_at);
        setRequests(combined);
      } finally {
        setLoading(false);
      }
    };

    fetchAllSelectedRequests();
  }, [selectedStatus]);

  const formatTimeRange = (timestamp: number) => {
    return format(new Date(timestamp), "MM-dd HH:mm:ss");
  };

  const formatDurationHuman = (start: number, end: number) => {
    const duration = intervalToDuration({
      start: new Date(start),
      end: new Date(end),
    });
    return formatDuration(duration, {
      format: ["years", "months", "weeks", "days", "hours", "minutes"],
      zero: false,
    });
  };

  const formatCreatedAt = (timestamp: number) => {
    const date = new Date(timestamp);
    return formatDistanceToNow(date, { addSuffix: true });
  };

  return (
    <div className="px-4 py-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Delete Requests
        </h1>
        <Link
          to="/deletes/new"
          className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-offset-gray-800"
        >
          New Delete Request
        </Link>
      </div>

      {error && (
        <div className="rounded-md bg-red-50 dark:bg-red-900 p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg
                className="h-5 w-5 text-red-400"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-red-800 dark:text-red-200">
                {error}
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="flex items-center space-x-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          Status:
        </h2>
        <div className="flex items-center">
          <button
            onClick={() => setSelectedStatus(["received"])}
            className={`px-4 py-2 text-sm font-medium rounded-l-md border ${
              selectedStatus.includes("received")
                ? "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900 dark:text-blue-200 dark:border-blue-800"
                : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"
            }`}
          >
            Received
          </button>
          <button
            onClick={() => setSelectedStatus(["processed"])}
            className={`px-4 py-2 text-sm font-medium rounded-r-md border-t border-r border-b -ml-px ${
              selectedStatus.includes("processed")
                ? "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900 dark:text-blue-200 dark:border-blue-800"
                : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"
            }`}
          >
            Processed
          </button>
        </div>
      </div>

      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
          <div className="shadow overflow-hidden border-b border-gray-200 dark:border-gray-700 sm:rounded-lg">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th
                    scope="col"
                    className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-20"
                  >
                    Status
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-20"
                  >
                    User
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-32"
                  >
                    Created At
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-40"
                  >
                    Time Range
                  </th>
                  <th
                    scope="col"
                    className="px-2 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24"
                  >
                    Duration
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-20"
                  >
                    Deleted Lines
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                  >
                    Query
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {loading ? (
                  <tr>
                    <td colSpan={7} className="px-6 py-4 text-center">
                      Loading...
                    </td>
                  </tr>
                ) : requests.length === 0 ? (
                  <tr>
                    <td
                      colSpan={7}
                      className="px-6 py-4 text-center text-gray-500 dark:text-gray-400"
                    >
                      No delete requests found
                    </td>
                  </tr>
                ) : (
                  requests.map((request, index) => (
                    <tr key={index}>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                            request.status === "processed"
                              ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                              : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                          }`}
                        >
                          {request.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                        {request.user_id}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                        <DateWithHover date={new Date(request.created_at)}>
                          {formatCreatedAt(request.created_at)}
                        </DateWithHover>
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-900 dark:text-gray-100">
                        <div className="space-y-0.5">
                          <div className="text-xs text-gray-500 dark:text-gray-400 uppercase font-medium">
                            From
                          </div>
                          <div className="font-mono text-xs pl-2">
                            {formatTimeRange(request.start_time)}
                          </div>
                          <div className="text-xs text-gray-500 dark:text-gray-400 uppercase font-medium pt-1">
                            To
                          </div>
                          <div className="font-mono text-xs pl-2">
                            {formatTimeRange(request.end_time)}
                          </div>
                        </div>
                      </td>
                      <td className="px-2 py-3 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                        {formatDurationHuman(
                          request.start_time,
                          request.end_time
                        )}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                        {request.deleted_lines.toLocaleString()}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-100 font-mono break-all pr-6">
                        <div className="max-w-2xl whitespace-pre-wrap">
                          {request.query}
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Deletes;
