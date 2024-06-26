// StatusComponent.tsx
import React, { useState, useEffect } from 'react';
import { Card, Tracker, type Color } from '@tremor/react';
import moment from 'moment';

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

interface Tracker { color: Color; tooltip: string; }

const mapStatusToTrackerData = (statuses: StatusData[]): Tracker[] => {
    return statuses.map(status => {
        let color: Color = 'emerald';
        let time = moment(status.time).format('MMMM Do YYYY, h:mm:ss a')
        let tooltip = `${time}`;

        if (status.status === 0) {
            color = 'rose';
            tooltip = 'Downtime';
        } else if (status.ping > 1000) { // Assuming ping > 1000ms is considered degraded
            color = 'yellow';
            tooltip = 'Degraded';
        }

        return { color, tooltip };
    });
};


const StatusComponent: React.FC = () => {
    const [statusData, setStatusData] = useState<ResponseData | null>(null);

    useEffect(() => {
        const eventSource = new EventSource('http://localhost:8080/status');

        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data) as ResponseData;

            console.log(statusData)
            setStatusData(data);
        };

        eventSource.onerror = (error) => {
            console.error('SSE Error:', error);
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
        <div className="bg-[#080B0F] text-white min-h-screen">
            <h1>Status Page</h1>
            {Object.entries(statusData.siteStatuses).map(([key, siteStatus]) => (
                <div key={key}>
                    <Card className="mx-auto max-w-md">
                        <p className="text-tremor-default flex items-center justify-between">
                            <span className="text-tremor-content-strong dark:text-dark-tremor-content-strong font-medium">{siteStatus.site.url}</span>
                            <span className="text-tremor-content dark:text-dark-tremor-content">uptime {Math.round(siteStatus.uptime)}%</span>
                        </p>
                        <Tracker data={mapStatusToTrackerData(siteStatus.statuses)} className="mt-2" />
                    </Card>
                </div>
            ))}
        </div>
    );
};

export default StatusComponent;
