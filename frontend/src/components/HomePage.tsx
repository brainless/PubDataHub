import React, { useState, useEffect } from 'react';
import { homeAPI } from '../services/api';

const HomePage: React.FC = () => {
  const [homePath, setHomePath] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>('');

  useEffect(() => {
    const fetchHomeDirectory = async () => {
      try {
        setLoading(true);
        setError('');
        const response = await homeAPI.getHomeDirectory();
        setHomePath(response.homePath);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to fetch home directory';
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    fetchHomeDirectory();
  }, []);

  return (
    <div style={{ padding: '2rem', fontFamily: 'Arial, sans-serif' }}>
      <h1 style={{ color: '#333', marginBottom: '2rem' }}>PubDataHub</h1>
      
      <div style={{ marginBottom: '1rem' }}>
        <h2 style={{ color: '#666', fontSize: '1.2rem', marginBottom: '0.5rem' }}>
          Home Directory
        </h2>
        
        {loading && (
          <div style={{ color: '#666', fontStyle: 'italic' }}>
            Loading your home directory...
          </div>
        )}
        
        {error && (
          <div style={{ 
            color: '#d32f2f', 
            backgroundColor: '#ffebee', 
            padding: '1rem', 
            borderRadius: '4px',
            border: '1px solid #ffcdd2'
          }}>
            <strong>Error:</strong> {error}
          </div>
        )}
        
        {!loading && !error && homePath && (
          <div style={{ 
            backgroundColor: '#e8f5e8', 
            padding: '1rem', 
            borderRadius: '4px',
            border: '1px solid #c8e6c9'
          }}>
            <strong>Your home directory:</strong> 
            <code style={{ 
              marginLeft: '0.5rem', 
              backgroundColor: '#f5f5f5', 
              padding: '0.25rem 0.5rem',
              borderRadius: '3px',
              fontFamily: 'monospace'
            }}>
              {homePath}
            </code>
          </div>
        )}
      </div>
    </div>
  );
};

export default HomePage;