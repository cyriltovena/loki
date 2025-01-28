import { useState, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import DatePicker from "react-datepicker";
import "react-datepicker/dist/react-datepicker.css";
import "../styles/datepicker.css";
import debounce from "lodash/debounce";
import { format, formatDuration, intervalToDuration } from "date-fns";

interface FormData {
  query: string;
  start_time: Date;
  end_time: Date;
  tenant_id: string;
}

interface FormErrors {
  tenant_id?: string;
  query?: string;
  start_time?: string;
  end_time?: string;
}

const NewDeleteRequest = () => {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [queryValidating, setQueryValidating] = useState(false);
  const [formData, setFormData] = useState<FormData>({
    query: "",
    start_time: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // 1 week ago
    end_time: new Date(), // now
    tenant_id: "",
  });

  const [errors, setErrors] = useState<FormErrors>({});

  const validateQuery = useCallback(
    async (query: string, shouldFormat = false) => {
      if (!query.trim()) {
        setErrors((prev) => ({ ...prev, query: "Query is required" }));
        return;
      }

      setQueryValidating(true);
      try {
        const response = await fetch(
          `/compactor/ui/api/format_query?query=${query}`,
          {
            method: "POST",
          }
        );

        const result = await response.json();

        if (!response.ok || result.status === "invalid-query") {
          throw new Error(result.error || "Invalid LogQL query");
        }

        // Only update the query with formatted version when shouldFormat is true
        if (shouldFormat) {
          setFormData((prev) => ({
            ...prev,
            query: result.data,
          }));
        }
        // Clear any existing query error
        setErrors((prev) => ({ ...prev, query: undefined }));
      } catch (error) {
        setErrors((prev) => ({
          ...prev,
          query: error instanceof Error ? error.message : "Invalid LogQL query",
        }));
      } finally {
        setQueryValidating(false);
      }
    },
    [setErrors, setFormData, setQueryValidating]
  );

  // Debounced validation function for typing with validateQuery in dependencies
  const debouncedValidate = useMemo(
    () => debounce((query: string) => validateQuery(query, false), 1000),
    [validateQuery]
  );

  const isValid =
    !errors.query &&
    formData.tenant_id.trim() &&
    formData.end_time > formData.start_time;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTouched({
      tenant_id: true,
      query: true,
      start_time: true,
      end_time: true,
    });

    // Basic validation before submission
    if (!formData.tenant_id.trim()) {
      setErrors((prev) => ({ ...prev, tenant_id: "Tenant ID is required" }));
      return;
    }
    if (formData.end_time <= formData.start_time) {
      setErrors((prev) => ({
        ...prev,
        end_time: "End time must be after start time",
      }));
      return;
    }
    if (!isValid) return;

    setError(null);
    setLoading(true);
    const params = new URLSearchParams();
    params.append("query", formData.query);
    params.append(
      "start",
      Math.floor(formData.start_time.getTime() / 1000).toString()
    );
    params.append(
      "end",
      Math.floor(formData.end_time.getTime() / 1000).toString()
    );
    try {
      const response = await fetch(
        `/compactor/ui/api/deletes?${params.toString()}`,
        {
          method: "POST",
          headers: {
            "X-Scope-OrgID": formData.tenant_id,
          },
        }
      );

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to create delete request");
      }

      navigate("/deletes");
    } catch (error) {
      console.error("Error creating delete request:", error);
      setError(
        error instanceof Error
          ? error.message
          : "Failed to create delete request"
      );
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
    setTouched((prev) => ({ ...prev, [name]: true }));

    // Clear tenant_id error when typing
    if (name === "tenant_id" && value.trim()) {
      setErrors((prev) => ({ ...prev, tenant_id: undefined }));
    }

    // Trigger debounced validation for query changes
    if (name === "query") {
      debouncedValidate(value);
    }
  };

  const handleBlur = async (field: string) => {
    setTouched((prev) => ({ ...prev, [field]: true }));

    // Validate and format LogQL query when the query field loses focus
    if (field === "query" && formData.query.trim()) {
      await validateQuery(formData.query, true);
    }

    // Validate tenant_id on blur
    if (field === "tenant_id" && !formData.tenant_id.trim()) {
      setErrors((prev) => ({ ...prev, tenant_id: "Tenant ID is required" }));
    }
  };

  const inputStyles = `block w-full px-4 py-3.5 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md text-gray-900 dark:text-gray-200 text-base focus:outline-none focus:ring-2 focus:ring-blue-500`;
  const errorInputStyles = `block w-full px-4 py-3.5 bg-red-50 dark:bg-red-900/50 border border-red-300 dark:border-red-700 rounded-md text-red-900 dark:text-red-200 text-base focus:outline-none focus:ring-2 focus:ring-red-500`;

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-8">
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg overflow-hidden">
          <form className="space-y-8 p-8" onSubmit={handleSubmit}>
            <div className="border-b border-gray-200 dark:border-gray-700 pb-4">
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                New Delete Request
              </h1>
            </div>

            {error && (
              <div className="rounded-md bg-red-50 dark:bg-red-900/50 p-4 border border-red-300 dark:border-red-700">
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

            <div className="space-y-2">
              <label
                className="block text-sm font-medium text-gray-700 dark:text-gray-200"
                htmlFor="tenant_id"
              >
                TENANT ID
              </label>
              <input
                type="text"
                name="tenant_id"
                id="tenant_id"
                required
                value={formData.tenant_id}
                onChange={handleChange}
                onBlur={() => handleBlur("tenant_id")}
                className={
                  touched.tenant_id && errors.tenant_id
                    ? errorInputStyles
                    : inputStyles
                }
                placeholder="Enter tenant ID"
              />
              {touched.tenant_id && errors.tenant_id && (
                <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                  {errors.tenant_id}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <label
                className="block text-sm font-medium text-gray-700 dark:text-gray-200"
                htmlFor="query"
              >
                LOGQL QUERY
              </label>
              <div className="relative">
                <textarea
                  name="query"
                  id="query"
                  required
                  value={formData.query}
                  onChange={handleChange}
                  onBlur={() => handleBlur("query")}
                  rows={4}
                  className={`${
                    touched.query && errors.query
                      ? errorInputStyles
                      : inputStyles
                  } font-mono ${queryValidating ? "opacity-50" : ""}`}
                  placeholder='{app="example"}'
                />
                {queryValidating && (
                  <div className="absolute right-3 top-3">
                    <svg
                      className="animate-spin h-5 w-5 text-blue-500"
                      xmlns="http://www.w3.org/2000/svg"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      ></circle>
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                  </div>
                )}
              </div>
              {touched.query && errors.query ? (
                <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                  {errors.query}
                </p>
              ) : (
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  Enter a LogQL query with labels in curly braces
                </p>
              )}
            </div>

            <div className="grid grid-cols-3 gap-8">
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  START TIME
                </label>
                <DatePicker
                  selected={formData.start_time}
                  onChange={(date) => {
                    setFormData((prev) => ({
                      ...prev,
                      start_time: date || new Date(),
                    }));
                    setTouched((prev) => ({ ...prev, start_time: true }));
                  }}
                  onBlur={() => handleBlur("start_time")}
                  showTimeSelect
                  timeFormat="HH:mm"
                  timeIntervals={15}
                  dateFormat="yyyy-MM-dd HH:mm"
                  className={
                    touched.start_time && errors.start_time
                      ? errorInputStyles
                      : inputStyles
                  }
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  END TIME
                </label>
                <DatePicker
                  selected={formData.end_time}
                  onChange={(date) => {
                    setFormData((prev) => ({
                      ...prev,
                      end_time: date || new Date(),
                    }));
                    setTouched((prev) => ({ ...prev, end_time: true }));
                  }}
                  onBlur={() => handleBlur("end_time")}
                  showTimeSelect
                  timeFormat="HH:mm"
                  timeIntervals={15}
                  dateFormat="yyyy-MM-dd HH:mm"
                  className={
                    touched.end_time && errors.end_time
                      ? errorInputStyles
                      : inputStyles
                  }
                  minDate={formData.start_time}
                />
                {touched.end_time && errors.end_time && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                    {errors.end_time}
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  DURATION
                </label>
                <div className="h-[42px] flex items-center">
                  <span className="text-sm text-gray-700 dark:text-gray-300">
                    {formatDuration(
                      intervalToDuration({
                        start: formData.start_time,
                        end: formData.end_time,
                      }),
                      {
                        format: [
                          "years",
                          "months",
                          "weeks",
                          "days",
                          "hours",
                          "minutes",
                        ],
                        zero: false,
                      }
                    )}
                  </span>
                </div>
              </div>
            </div>

            <div className="flex justify-end space-x-3 pt-6 border-t border-gray-200 dark:border-gray-700">
              <button
                type="button"
                onClick={() => navigate("/deletes")}
                className="px-6 py-3 text-base font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading || !isValid}
                className="px-6 py-3 text-base font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? "Creating..." : "Create Delete Request"}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default NewDeleteRequest;
