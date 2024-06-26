// StatusComponent.tsx
import React, { useState, useEffect } from 'react';
import { Card, Tracker, type Color } from '@tremor/react';
import moment from 'moment';
import { AreaChart } from '@tremor/react';

interface StatusData {
    status: number;
    time: string;
    msg: string;
    ping: number;
}

interface Site {
    id: string;
    url: string;
    name: string;
}

interface SiteStatus {
    site: Site;
    statuses: StatusData[];
    uptime: number;
}

interface ResponseData {
    siteStatuses: {
        [key: string]: SiteStatus;
    };
}

interface Tracker {
    color: Color;
    tooltip: string;
}

const mapStatusToTrackerData = (statuses: StatusData[]): Tracker[] => {
    return statuses.map((status) => {
        let color: Color = 'emerald';
        let time = moment(status.time).format('MMMM Do YYYY, h:mm:ss a');
        let tooltip = `${time}`;

        if (status.status === 0) {
            color = 'rose';
            tooltip = 'Downtime: ' + time;
        } else if (status.ping > 1000) {
            // Assuming ping > 1000ms is considered degraded
            color = 'yellow';
            tooltip = 'Degraded: ' + time;
        }

        return { color, tooltip };
    });
};

const mapStatusToChartData = (statuses: StatusData[]) => {
    return statuses.map((status) => ({
        //date: new Date(status.time).toLocaleDateString(),
        date: moment(status.time).format('h:mm:ss a'),
        Status: status.status,
        Ping: status.ping,
    }));
};

const reverStatuses = (data: ResponseData): ResponseData => {
    Object.entries(data.siteStatuses).map(([key, siteStatus]) => {
        return siteStatus.statuses = siteStatus.statuses.reverse()
    })
    return data
}

const dataFormatter = (number: number) => `${number}ms`;

const StatusComponent: React.FC = () => {
    const [statusData, setStatusData] = useState<ResponseData | null>(null);
    const [areAllOperational, setAllOperational] = useState(false);

    useEffect(() => {
        const eventSource = new EventSource('http://localhost:8080/status');

        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data) as ResponseData;


            setStatusData(
                reverStatuses(data)
            );

            const allOperational = Object.values(data.siteStatuses).every(
                (site) => site.statuses[site.statuses.length - 1].status === 1
            );
            setAllOperational(allOperational);
        };

        eventSource.onerror = (error) => {
            eventSource.close();
        };

        return () => {
            eventSource.close();
        };
    }, []);

    if (!statusData) {
        return <div>Loading status data...</div>;
    }

    return (
        <div className="bg-[#080B0F] text-white min-h-screen p-4 font-poppins">
            <section className="mx-auto max-w-4xl ">
                <h1 className="text-2xl font-bold mb-4">SENTINEL</h1>

                <div
                    className={`p-4 mb-6 rounded-sm text-start font-bold ${areAllOperational ? 'bg-green-500' : 'bg-red-500'
                        }`}
                >
                    {areAllOperational
                        ? 'All Systems Operational'
                        : 'Some Systems Are Down'}
                </div>

                {Object.entries(statusData.siteStatuses).map(
                    ([key, siteStatus]) => (
                        <div key={key} className="mb-8">
                            <Card className="mb-4">
                                <div className="mb-4">
                                    <p className="font-bold text-xl capitalize ">
                                        {siteStatus.site.name}
                                    </p>
                                </div>
                                <p className="text-tremor-default flex items-center justify-between mb-2">
                                    <span className="text-tremor-content-strong dark:text-dark-tremor-content-strong font-medium">
                                        {' '}
                                        <a
                                            target="_blank"
                                            href={siteStatus.site.url}
                                        >
                                            {siteStatus.site.url}
                                        </a>{' '}
                                    </span>
                                    <span className="text-tremor-content dark:text-dark-tremor-content">
                                        uptime {siteStatus.uptime.toFixed(2)}%
                                    </span>
                                </p>
                                <Tracker
                                    data={mapStatusToTrackerData(
                                        siteStatus.statuses
                                    )}
                                    className="mt-2 mb-4"
                                />
                                <AreaChart
                                    className="h-80"
                                    data={mapStatusToChartData(
                                        siteStatus.statuses
                                    )}
                                    index="date"
                                    categories={['Status', 'Ping']}
                                    colors={['indigo', 'rose']}
                                    valueFormatter={dataFormatter}
                                    yAxisWidth={60}
                                    onValueChange={(v) => console.log(v)}
                                />
                            </Card>
                        </div>
                    )
                )}
            </section>
        </div>
    );
};

export default StatusComponent;
